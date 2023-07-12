package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/amaurybrisou/ablib"
	"github.com/amaurybrisou/ablib/jwtlib"
	"github.com/amaurybrisou/ablib/mailcli"
	"github.com/amaurybrisou/ablib/store"
	"github.com/amaurybrisou/gateway/src"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/gwservices"
	"github.com/amaurybrisou/gateway/src/gwservices/payment"
	"github.com/amaurybrisou/gateway/src/gwservices/proxy"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	loglevel, err := zerolog.ParseLevel(ablib.LookupEnv("LOG_LEVEL", "debug"))
	if err != nil {
		fmt.Println("invalid LOG_LEVEL")
		os.Exit(1)
	}
	zerolog.SetGlobalLevel(loglevel)

	ablib.Logger(ablib.LookupEnv("LOG_FORMAT", "json"))

	ctx := log.Logger.WithContext(context.Background())

	defer func() {
		if r := recover(); r != nil {
			log.Ctx(ctx).Info().Any("recover", r).Send()
		}
	}()

	log.Ctx(ctx).Info().
		Any("build_version", src.BuildVersion).
		Any("build_hash", src.BuildHash).
		Any("build_time", src.BuildTime).
		Send()

	postgres := store.NewPostgres(ctx,
		ablib.LookupEnv("DB_USERNAME", "gateway"),
		ablib.LookupEnv("DB_PASSWORD", "gateway"),
		ablib.LookupEnv("DB_HOST", "localhost"),
		ablib.LookupEnvInt("DB_PORT", 5432),
		ablib.LookupEnv("DB_DATABASE", "gateway"),
		ablib.LookupEnv("DB_SSL_MODE", "disable"),
	)

	db := database.New(postgres)

	domain := ablib.LookupEnv("DOMAIN", "http://localhost:8089")

	mail, err := mailcli.NewMailClient(
		ctx,
		mailcli.WithMailClientOptionSenderEmail(ablib.LookupEnv("SENDER_EMAIL", "gateway@gateway.org")),
		mailcli.WithMailClientOptionSenderPassword(ablib.LookupEnv("SENDER_PASSWORD", "default-password")),
	)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Msg("creating mail client")
		return
	}

	services := gwservices.NewServices(db, mail, gwservices.ServiceConfig{
		PaymentConfig: payment.Config{
			StripeKey:           ablib.LookupEnv("STRIPE_KEY", ""),
			StripeSuccessURL:    ablib.LookupEnv("STRIPE_SUCCESS_URL", domain+"/login"),
			StripeCancelURL:     ablib.LookupEnv("STRIPE_CANCEL_URL", domain),
			StripeWebHookSecret: ablib.LookupEnv("STRIPE_WEBHOOK_SECRET", ""),
		},
		JwtConfig: jwtlib.Config{
			SecretKey: ablib.LookupEnv("JWT_KEY", "insecure-key"),
			Issuer:    ablib.LookupEnv("JWT_ISSUER", domain),
			Audience:  ablib.LookupEnv("JWT_AUDIENCE", "insecure-key"),
		},
		ProxyConfig: proxy.Config{
			StripPrefix:         "",
			NotFoundRedirectURL: "/services",
			NoRoleRedirectURL:   "/pricing",
		},
	})

	r := src.Router(services, db)

	lcore := ablib.NewCore(
		ablib.WithMigrate(
			ablib.LookupEnv("DB_MIGRATIONS_PATH", "file://migrations"),
			postgres.Config().ConnString(),
		),
		ablib.WithLogLevel(ablib.LookupEnv("LOG_LEVEL", "debug")),
		ablib.WithHTTPServer(
			ablib.LookupEnv("HTTP_SERVER_ADDR", "0.0.0.0"),
			ablib.LookupEnvInt("HTTP_SERVER_PORT", 8089),
			r,
		),
		ablib.WithSignals(),
		ablib.WithPrometheus(
			ablib.LookupEnv("HTTP_PROM_ADDR", "0.0.0.0"),
			ablib.LookupEnvInt("HTTP_PROM_PORT", 2112),
		),
		ablib.HeartBeat(
			ablib.WithRequestPath("/healthcheck"),
			ablib.WithClientTimeout(5*time.Second),
			ablib.WithInterval(ablib.LookupEnvDuration("HEARTBEAT_INTERVAL", "10s")),
			ablib.WithErrorIncrement(ablib.LookupEnvDuration("HEARTBEAT_ERROR_INCREMENT", "5s")),
			ablib.WithFetchServiceFunction(func(ctx context.Context) ([]ablib.Service, error) {
				services, err := db.GetServices(ctx)
				if err != nil {
					return nil, nil
				}
				output := make([]ablib.Service, len(services))
				for i, s := range services {
					output[i] = s
				}
				return output, nil
			}),
			ablib.WithUpdateServiceStatusFunction(func(ctx context.Context, u uuid.UUID, s string) error {
				return db.UpdateServiceStatus(ctx, u, s)
			}),
		),
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	started, errChan := lcore.Start(ctx)

	go func() {
		<-started
		log.Ctx(ctx).Debug().Msg("all backend services started")
	}()

	err = <-errChan
	if err != nil {
		if !errors.Is(err, ablib.ErrSignalReceived) {
			log.Ctx(ctx).Error().Err(err).Msg("error received")
		}
		err = lcore.Shutdown(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("shutdown error received")
		}
		log.Ctx(ctx).Debug().Msg("services stopped")
	}

	log.Ctx(ctx).Debug().Msg("shutdown")
}
