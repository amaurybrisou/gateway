package test

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/amaurybrisou/ablib"
	"github.com/amaurybrisou/ablib/jwtlib"
	"github.com/amaurybrisou/gateway/src"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/gwservices"
	"github.com/amaurybrisou/gateway/src/gwservices/payment"
	"github.com/amaurybrisou/gateway/src/gwservices/proxy"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/stripe/stripe-go/v72/webhook"
)

type DefaultTestSuite struct {
	suite.Suite
	lcore *ablib.Core

	Container  *Container
	DB         *database.Database
	connString string
}

func (s *DefaultTestSuite) SetupSuite() {
	ctx := log.Logger.WithContext(context.Background())
	// Pass `TEST_LOG_LEVEL=debug` for more logs
	var testLogLevel string
	if v := os.Getenv("TEST_LOG_LEVEL"); len(v) > 0 {
		testLogLevel = v
	} else {
		testLogLevel = "info"
	}
	testLogLevel = strings.ToUpper(testLogLevel)

	// Set up the Docker client and start the database container.
	var err error
	s.Container, err = NewContainer(
		nil,
		ContainerConfig{
			Repository: "postgres",
			Tag:        "latest",
			Env: []string{
				"POSTGRES_USER=gw",
				"POSTGRES_PASSWORD=gw",
				"POSTGRES_DB=gw",
			},
			Cmd: []string{"postgres", "-c", "log_statement=all", "-c", "log_destination=stderr"},
		},
	)
	require.NoError(s.T(), err)

	// Wait for the container to be ready and get its connection details.
	err = s.Container.Retry(
		func() error {
			s.connString = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", "gw", "gw", "localhost", s.Container.GetPort("5432/tcp"), "gw", "disable")
			db, err := pgxpool.New(ctx, s.connString)
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Send()
				return err
			}

			err = db.Ping(ctx)
			if err != nil {
				return err
			}

			s.DB = database.New(db)
			s.connString = db.Config().ConnString()

			return nil
		},
	)
	require.NoError(s.T(), err)

	if testLogLevel == "DEBUG" {
		go func() {
			err = s.Container.TailLogs(ctx, os.Stdout, true)
			require.NoError(s.T(), err)
		}()
	}

	domain := ablib.LookupEnv("DOMAIN", "http://localhost:50000")

	services := gwservices.NewServices(s.DB, nil, gwservices.ServiceConfig{
		PaymentConfig: payment.Config{
			StripeKey:           ablib.LookupEnv("STRIPE_KEY", ""),
			StripeSuccessURL:    ablib.LookupEnv("STRIPE_SUCCESS_URL", domain+"/login"),
			StripeCancelURL:     ablib.LookupEnv("STRIPE_CANCEL_URL", domain),
			StripeWebHookSecret: ablib.LookupEnv("STRIPE_WEBHOOK_SECRET", "test-webhook-secret"),
		},
		JwtConfig: jwtlib.Config{
			SecretKey: ablib.LookupEnv("JWT_KEY", "insecure-key"),
			Issuer:    ablib.LookupEnv("JWT_ISSUER", domain),
			Audience:  ablib.LookupEnv("JWT_AUDIENCE", "insecure-key"),
		},
		ProxyConfig: proxy.Config{
			StripPrefix:         "/auth",
			NotFoundRedirectURL: "/services",
			NoRoleRedirectURL:   "/pricing",
		},
	})

	r := src.Router(services, s.DB, "", 10, 10)

	lcore := ablib.NewCore(
		ablib.WithMigrate(
			ablib.LookupEnv("DB_MIGRATIONS_PATH", "file://../migrations"),
			s.connString,
		),
		ablib.WithLogLevel(ablib.LookupEnv("LOG_LEVEL", "debug")),
		ablib.WithHTTPServer(
			ablib.LookupEnv("HTTP_SERVER_ADDR", "0.0.0.0"),
			50000,
			r,
		),
	)

	s.lcore = lcore

	ctx, cancel := context.WithCancel(ctx)

	startedChan, hasErrorChan := s.lcore.Start(ctx)

	go func() {
		err := <-hasErrorChan
		if err != nil {
			log.Ctx(ctx).Debug().Caller().Err(err).Msg("closing services...")
			s.lcore.Shutdown(ctx) //nolint
		}

		defer cancel()
	}()

	<-startedChan
}

func (s *DefaultTestSuite) TearDownSuite() {
	ctx := log.Logger.WithContext(context.Background())
	s.lcore.Shutdown(ctx) //nolint
	// Stop the database container and clean up resources.
	err := s.Container.Purge(s.Container.Resource)
	assert.NoError(s.T(), err)
}

func (s *DefaultTestSuite) Post(path, contentType, body string) (*http.Response, error) {
	return http.Post("http://localhost:50000"+path, contentType, strings.NewReader(body))
}

func (s *DefaultTestSuite) ReadFile(path string) []byte {
	b, err := os.ReadFile(path)
	require.NoError(s.T(), err)
	return b
}

func (s *DefaultTestSuite) PostWebhook(contentType string, bodyBytes []byte) (*http.Response, error) {
	now := time.Now()
	signature := webhook.ComputeSignature(now, bodyBytes, "test-webhook-secret")
	req, err := http.NewRequest(http.MethodPost, "http://localhost:50000/payment/webhook", bytes.NewReader(bodyBytes))
	require.NoError(s.T(), err)
	req.Header.Add("Stripe-Signature", fmt.Sprintf("t=%d,v1=%s", now.Unix(), hex.EncodeToString(signature)))
	return http.DefaultClient.Do(req)
}
