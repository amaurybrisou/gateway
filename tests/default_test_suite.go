package test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/amaurybrisou/gateway/pkg/core"
	"github.com/amaurybrisou/gateway/pkg/core/jwtlib"
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
)

type DefaultTestSuite struct {
	suite.Suite
	lcore *core.Core

	Container  *Container
	db         *database.Database
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

			s.db = database.New(db)
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

	domain := core.LookupEnv("DOMAIN", "http://localhost:50000")

	services := gwservices.NewServices(s.db, gwservices.ServiceConfig{
		PaymentConfig: payment.Config{
			StripeKey:           core.LookupEnv("STRIPE_KEY", ""),
			StripeSuccessURL:    core.LookupEnv("STRIPE_SUCCESS_URL", domain+"/login"),
			StripeCancelURL:     core.LookupEnv("STRIPE_CANCEL_URL", domain),
			StripeWebHookSecret: core.LookupEnv("STRIPE_WEBHOOK_SECRET", ""),
		},
		JwtConfig: jwtlib.Config{
			SecretKey: core.LookupEnv("JWT_KEY", "insecure-key"),
			Issuer:    core.LookupEnv("JWT_ISSUER", domain),
			Audience:  core.LookupEnv("JWT_AUDIENCE", "insecure-key"),
		},
		ProxyConfig: proxy.Config{
			StripPrefix:         "/auth",
			NotFoundRedirectURL: "/services",
			NoRoleRedirectURL:   "/pricing",
		},
	})

	r := src.Router(services, s.db)

	lcore := core.New(
		core.WithMigrate(
			core.LookupEnv("DB_MIGRATIONS_PATH", "file://../migrations"),
			s.connString,
		),
		core.WithLogLevel(core.LookupEnv("LOG_LEVEL", "debug")),
		core.WithHTTPServer(
			core.LookupEnv("HTTP_SERVER_ADDR", "0.0.0.0"),
			50000,
			r,
		),
	)

	s.lcore = lcore

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	startedChan, hasErrorChan := lcore.Start(ctx)

	<-startedChan

	go func() {
		for err := range hasErrorChan {
			cancel()

			if errors.Is(err, core.ErrSignalReceived) {
				log.Ctx(ctx).Debug().Caller().Err(err).Msg("closing services...")
			}

			lcore.Shutdown(ctx)
		}
	}()
}

func (s *DefaultTestSuite) TearDownSuite() {
	ctx := log.Logger.WithContext(context.Background())
	s.lcore.Shutdown(ctx)
	// Stop the database container and clean up resources.
	err := s.Container.Purge(s.Container.Resource)
	assert.NoError(s.T(), err)
}

func (s *DefaultTestSuite) Post(path, contentType, body string) (*http.Response, error) {
	return http.Post("http://localhost:50000"+path, contentType, strings.NewReader(body))
}
