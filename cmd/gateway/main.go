package main

import (
	"context"
	"time"

	"github.com/amaurybrisou/gateway/pkg/core"
	"github.com/amaurybrisou/gateway/pkg/core/jwtlib"
	"github.com/amaurybrisou/gateway/pkg/core/store"
	"github.com/amaurybrisou/gateway/src"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/gwservices"
	"github.com/amaurybrisou/gateway/src/gwservices/payment"
	"github.com/amaurybrisou/gateway/src/gwservices/proxy"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func main() {
	core.Logger()
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
		core.LookupEnv("DB_USERNAME", "gateway"),
		core.LookupEnv("DB_PASSWORD", "gateway"),
		core.LookupEnv("DB_HOST", "localhost"),
		core.LookupEnvInt("DB_PORT", 5432),
		core.LookupEnv("DB_DATABASE", "gateway"),
		core.LookupEnv("DB_SSL_MODE", "disable"),
	)

	db := database.New(postgres)

	domain := core.LookupEnv("DOMAIN", "http://localhost:8089")

	services := gwservices.NewServices(db, gwservices.ServiceConfig{
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

	r := src.Router(services, db)

	lcore := core.New(
		core.WithMigrate(
			core.LookupEnv("DB_MIGRATIONS_PATH", "file://migrations"),
			postgres.Config().ConnString(),
		),
		core.WithLogLevel(core.LookupEnv("LOG_LEVEL", "debug")),
		core.WithHTTPServer(
			core.LookupEnv("HTTP_SERVER_ADDR", "0.0.0.0"),
			core.LookupEnvInt("HTTP_SERVER_PORT", 8089),
			r,
		),
		core.WithPrometheus(
			core.LookupEnv("HTTP_PROM_ADDR", "0.0.0.0"),
			core.LookupEnvInt("HTTP_PROM_PORT", 2112),
		),
		core.HeartBeat(
			core.WithRequestPath("/hc"),
			core.WithClientTimeout(5*time.Second),
			core.WithInterval(core.LookupEnvDuration("HEARTBEAT_INTERVAL", "10s")),
			core.WithErrorIncrement(core.LookupEnvDuration("HEARTBEAT_ERROR_INCREMENT", "5s")),
			core.WithFetchServiceFunction(func(ctx context.Context) ([]core.Service, error) {
				services, err := db.GetServices(ctx)
				if err != nil {
					return nil, nil
				}
				output := make([]core.Service, len(services))
				for i, s := range services {
					output[i] = s
				}
				return output, nil
			}),
			core.WithUpdateServiceStatusFunction(func(ctx context.Context, u uuid.UUID, s string) error {
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

	err := <-errChan
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("shutting down")
		lcore.Shutdown(ctx)
		log.Ctx(ctx).Debug().Msg("services stopped")
	}

	log.Ctx(ctx).Debug().Msg("shutdown")
}
