package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/amaurybrisou/gateway/pkg/core"
	"github.com/amaurybrisou/gateway/pkg/core/jwtlib"
	coremodels "github.com/amaurybrisou/gateway/pkg/core/models"
	"github.com/amaurybrisou/gateway/pkg/core/store"
	coremiddleware "github.com/amaurybrisou/gateway/pkg/http/middleware"
	"github.com/amaurybrisou/gateway/src"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/gwservices"
	"github.com/amaurybrisou/gateway/src/gwservices/payment"
	"github.com/amaurybrisou/gateway/src/gwservices/proxy"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func main() {
	core.Logger()
	ctx := log.Logger.WithContext(context.Background())

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

	// google.New(
	services := gwservices.NewServices(db, gwservices.ServiceConfig{
		PaymentConfig: payment.Config{
			StripeKey:           core.LookupEnv("STRIPE_KEY", ""),
			StripeSuccessURL:    core.LookupEnv("STRIPE_SUCCESS_URL", "http://localhost:8089/login"),
			StripeCancelURL:     core.LookupEnv("STRIPE_CANCEL_URL", "http://localhost:8089"),
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

	r := router(services, db)

	heartbeatInterval, err := time.ParseDuration(core.LookupEnv("HEARTBEAT_INTERVAL", "2s"))
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Send()
	}

	heartbeatErrorIncrement, err := time.ParseDuration(core.LookupEnv("HEARTBEAT_ERROR_INCREMENT", "5s"))
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Send()
	}

	err = core.Run(
		ctx,
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
			core.WithRequestPath("/healthcheck"),
			core.WithClientTimeout(5*time.Second),
			core.WithInterval(heartbeatInterval),
			core.WithErrorIncrement(heartbeatErrorIncrement),
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

	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Msg("shutting down")
	}
}

func router(s gwservices.Services, db *database.Database) http.Handler {
	r := chi.NewRouter()

	r.Use(coremiddleware.NewRateLimitMiddleware(coremiddleware.WithRateLimit(5, 10)).Middleware)
	r.Use(coremiddleware.RequestMetric("gateway"))
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.RequestLogger(core.Logger()))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(time.Second * 10))

	// UNAUTHENTICATED
	// r.HandleFunc("/", rootHandler(db))
	r.Post("/login", s.Service().LoginHandler)
	r.Post("/payment/webhook", s.Payment().StripeWebhook)
	r.With(coremiddleware.JsonContentType()).Get("/services", s.Service().GetAllServicesHandler)
	r.Get("/pricing/{service_name}", s.Service().ServicePricePage)

	// AUTHENTICATED
	authMiddleware := coremiddleware.NewAuthMiddleware(s.Jwt(), func(ctx context.Context, id uuid.UUID) (coremodels.UserInterface, error) {
		return db.GetUserByID(ctx, id)
	})

	r.Route("/auth", func(authenticatedRouter chi.Router) {
		authenticatedRouter.Use(authMiddleware.JWTAuth)

		authenticatedRouter.Route("/admin", func(adminRouter chi.Router) {
			adminRouter.Use(authMiddleware.IsAdmin)
			adminRouter.Use(coremiddleware.JsonContentType())

			adminRouter.Post("/services", s.Service().CreateServiceHandler)
			adminRouter.Delete("/services/{service_id}", s.Service().DeleteServiceHandler)
			adminRouter.Get("/services", s.Service().GetAllServicesHandler)
			adminRouter.Get("/version", src.Version)
		})

		authenticatedRouter.HandleFunc("/*", s.Proxy().ProxyHandler)
	})

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		route = strings.Replace(route, "/*/", "/", -1)
		log.Debug().
			Any("method", method).
			Any("route", route).
			Send()
		return nil
	}

	if err := chi.Walk(r, walkFunc); err != nil {
		fmt.Printf("Logging err: %s\n", err.Error())
	}

	return r
}
