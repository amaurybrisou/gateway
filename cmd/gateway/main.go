package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/amaurybrisou/gateway/pkg/core"
	"github.com/amaurybrisou/gateway/pkg/core/jwtlib"
	"github.com/amaurybrisou/gateway/pkg/core/store"
	coremiddleware "github.com/amaurybrisou/gateway/pkg/http/middleware"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/amaurybrisou/gateway/src/gwservices"
	"github.com/amaurybrisou/gateway/src/gwservices/payment"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

var (
	BuildVersion, BuildHash, BuildTime string = "1.0", "localhost", time.Now().String()
)

func main() {
	core.Logger()

	ctx := log.Logger.WithContext(context.Background())
	dbUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		core.LookupEnv("DB_USERNAME", "gateway"),
		core.LookupEnv("DB_PASSWORD", "gateway"),
		core.LookupEnv("DB_HOST", "localhost"),
		core.LookupEnvInt("DB_PORT", 5432),
		core.LookupEnv("DB_DATABASE", "gateway"),
	)

	mig := core.WithMigrate(
		core.LookupEnv("DB_MIGRATIONS_PATH", "file://migrations"),
		dbUrl,
	)

	if err := mig.Start(ctx); err != nil {
		log.Ctx(ctx).Fatal().Err(err).Send()
	}

	postgres, err := store.NewPostgres(ctx, dbUrl)
	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Send()
	}

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
	})

	r := router(services, db)

	err = core.Run(
		ctx,
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
	)

	if err != nil {
		log.Ctx(ctx).Fatal().Err(err).Msg("shutting down")
	}
}

func router(s gwservices.Services, db *database.Database) http.Handler {
	r := mux.NewRouter()

	r.Use(coremiddleware.RequestMetric("gateway"))
	r.Use(coremiddleware.Logger(log.Logger))
	r.Use(coremiddleware.JsonContentType())

	authMiddleware := coremiddleware.NewAuthMiddleware(db, s.Jwt())

	// UNAUTHENTICATED
	// r.HandleFunc("/", rootHandler(db))
	r.HandleFunc("/login", s.Service().LoginHandler)
	r.HandleFunc("/payment/webhook", s.Payment().StripeWebhook)
	r.HandleFunc("/services", s.Service().GetAllServicesHandler).Methods(http.MethodGet)
	r.HandleFunc("/pricing/{service_name}", s.Service().ServicePricePage).Methods(http.MethodGet)

	// AUTHENTICATED
	authenticatedRouter := r.NewRoute().Subrouter()
	authenticatedRouter.Use(authMiddleware.JWTAuth)

	adminRouter := authenticatedRouter.NewRoute().Subrouter()
	adminRouter.Use(func(h http.Handler) http.Handler {
		return authMiddleware.IsAdmin(h)
	})

	adminRouter.HandleFunc("/services", s.Service().CreateServiceHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/services", s.Service().DeleteServiceHandler).Methods(http.MethodDelete)

	authenticatedRouter.PathPrefix("/").Handler(s.Proxy().ProxyHandler(rootHandler(db)))

	return r
}

func rootHandler(db *database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		servicesList, err := db.GetServices(r.Context())
		if err != nil {
			log.Ctx(r.Context()).Err(err).Send()
			http.Error(w, "fetch services", http.StatusInternalServerError)
			return
		}

		userInt := coremiddleware.User(r.Context())
		if userInt == nil {
			if err := json.NewEncoder(w).Encode(struct {
				Services []models.Service `json:"services"`
			}{
				Services: servicesList,
			}); err != nil {
				log.Ctx(r.Context()).Err(err).Send()
				http.Error(w, "error marshaling", http.StatusInternalServerError)
				return
			}
			return
		}

		user := models.NewUserFromInt(userInt)

		if err := json.NewEncoder(w).Encode(struct {
			User     models.User      `json:"user"`
			Services []models.Service `json:"services"`
		}{
			User:     user,
			Services: servicesList,
		}); err != nil {
			log.Ctx(r.Context()).Err(err).Send()
			http.Error(w, "error marshaling", http.StatusInternalServerError)
			return
		}
	}
}
