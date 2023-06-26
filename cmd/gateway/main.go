package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/amaurybrisou/gateway/internal/db"
	"github.com/amaurybrisou/gateway/internal/db/models"
	"github.com/amaurybrisou/gateway/internal/services"
	"github.com/amaurybrisou/gateway/pkg/core"
	"github.com/amaurybrisou/gateway/pkg/core/store"
	coremiddleware "github.com/amaurybrisou/gateway/pkg/http/middleware"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

func main() {
	core.Logger()

	// cfg := config.New()

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

	db := db.New(postgres)

	services := services.NewServices(db, services.ServiceConfig{
		StripeKey:        os.Getenv("STRIPE_KEY"),
		StripeSuccessURL: "http://localhost:8089/payment/success",
		StripeCancelURL:  "http://localhost:8089/payment/cancel",
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

func router(s services.Services, db *db.Database) http.Handler {

	r := mux.NewRouter()

	r.Use(coremiddleware.RequestMetric("gateway"))
	r.Use(coremiddleware.Logger(log.Logger))
	r.Use(coremiddleware.JsonContentType())

	authMiddleware := coremiddleware.NewAuthMiddleware(
		db,
		[]string{
			"/",
			"/auth/google",
			"/auth/google/callback",
			"/services",
			"/payment/create",
			"/payment/sucess",
		},
	)

	r.Use(func(h http.Handler) http.Handler { return authMiddleware.BearerAuth(h, db.GetUserByAccessToken) })

	// unauthenticated
	// r.HandleFunc("/", rootHandler(db))
	r.HandleFunc("/auth/{provider}", s.Oauth().AuthHandler)
	r.HandleFunc("/auth/{provider}/callback", s.Oauth().CallBackHandler)
	r.HandleFunc("/services", s.Service().GetAllServicesHandler).Methods(http.MethodGet)
	r.HandleFunc("/payment/create", s.Payment().BuyServiceHandler).Methods(http.MethodPost)
	r.HandleFunc("/payment/success", s.Payment().StripeSuccess)

	// require authentication
	r.HandleFunc("/logout/{provider}", s.Oauth().LogoutHandler)

	adminRouter := r.NewRoute().Subrouter()
	adminRouter.Use(func(h http.Handler) http.Handler {
		return authMiddleware.IsAdmin(h)
	})

	adminRouter.HandleFunc("/services", s.Service().CreateServiceHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/services", s.Service().DeleteServiceHandler).Methods(http.MethodDelete)

	r.PathPrefix("/").Handler(s.Proxy().ProxyHandler(rootHandler(db)))

	return r
}

func rootHandler(db *db.Database) http.HandlerFunc {
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
