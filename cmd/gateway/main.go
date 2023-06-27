package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/amaurybrisou/gateway/internal/db"
	"github.com/amaurybrisou/gateway/internal/db/models"
	"github.com/amaurybrisou/gateway/internal/services"
	"github.com/amaurybrisou/gateway/internal/services/oauth"
	"github.com/amaurybrisou/gateway/internal/services/payment"
	"github.com/amaurybrisou/gateway/pkg/core"
	"github.com/amaurybrisou/gateway/pkg/core/jwtlib"
	"github.com/amaurybrisou/gateway/pkg/core/store"
	coremiddleware "github.com/amaurybrisou/gateway/pkg/http/middleware"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
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

	db := db.New(postgres)

	// google.New(
	services := services.NewServices(db, services.ServiceConfig{
		PaymentConfig: payment.Config{
			StripeKey:           core.LookupEnv("STRIPE_KEY", ""),
			StripeSuccessURL:    core.LookupEnv("STRIPE_SUCCESS_URL", "http://localhost:8089/login"),
			StripeCancelURL:     core.LookupEnv("STRIPE_CANCEL_URL", "http://localhost:8089"),
			StripeWebHookSecret: core.LookupEnv("STRIPE_WEBHOOK_SECRET", ""),
		},
		GoogleConfig: oauth.Config{
			GoogleKey:         core.LookupEnv("GOOGLE_KEY", ""),
			GoogleSecret:      core.LookupEnv("GOOGLE_SECRET", ""),
			GoogleCallBackURL: core.LookupEnv("GOOGLE_CALLBACK_URL", "http://localhost:8089/auth/google/callback"),
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

func router(s services.Services, db *db.Database) http.Handler {

	r := mux.NewRouter()

	r.Use(coremiddleware.RequestMetric("gateway"))
	r.Use(coremiddleware.Logger(log.Logger))
	r.Use(coremiddleware.JsonContentType())

	authMiddleware := coremiddleware.NewAuthMiddleware(
		db,
		[]string{
			"/",
			// "/auth/{provider}",
			// "/auth/{provider}/callback",
			"/pricing/{service_name}",
			"/services",
			"/payment/create",
			"/payment/webhook",
			"/login",
		},
	)

	l := jwtlib.New("secret")
	// UNAUTHENTICATED
	// r.HandleFunc("/", rootHandler(db))
	r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		type Credentials struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		// Parse the request body into a Credentials struct
		var creds Credentials
		err := json.NewDecoder(r.Body).Decode(&creds)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Verify the credentials (example: hardcoded username and password)
		if creds.Username != "admin" || creds.Password != "password" {
			http.Error(w, "Invalid username or password", http.StatusUnauthorized)
			return
		}

		// Generate a JWT token with a subject and expiration time
		token, err := l.GenerateToken(creds.Username, time.Now().Add(time.Hour))
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Return the token as the response
		response := map[string]string{"token": token}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response) //nolint
	})
	// authenticatedRouter.HandleFunc("/payment/create", s.Payment().BuyServiceHandler).Methods(http.MethodPost)
	// r.HandleFunc("/auth/{provider}", s.Oauth().AuthHandler)
	// r.HandleFunc("/auth/{provider}/callback", s.Oauth().CallBackHandler)
	r.HandleFunc("/payment/webhook", s.Payment().StripeWebhook)
	r.HandleFunc("/services", s.Service().GetAllServicesHandler).Methods(http.MethodGet)
	r.HandleFunc("/pricing/{service_name}", s.Service().ServicePricePage).Methods(http.MethodGet)

	// AUTHENTICATED
	authenticatedRouter := r.NewRoute().Subrouter()

	// authenticatedRouter.Use(func(h http.Handler) http.Handler {
	// 	return authMiddleware.BearerAuth(http.RedirectHandler("/", http.StatusPermanentRedirect), db.GetUserByAccessToken)
	// })

	authenticatedRouter.Use(func(h http.Handler) http.Handler {
		return coremiddleware.JWTAuth(h, "secret")
	})

	// authenticatedRouter.Use(func(h http.Handler) http.Handler {
	// 	return authMiddleware.SessionAuth(h)
	// })

	// authenticatedRouter.HandleFunc("/logout/{provider}", s.Oauth().LogoutHandler)

	adminRouter := authenticatedRouter.NewRoute().Subrouter()
	adminRouter.Use(func(h http.Handler) http.Handler {
		return authMiddleware.IsAdmin(h)
	})

	adminRouter.HandleFunc("/services", s.Service().CreateServiceHandler).Methods(http.MethodPost)
	adminRouter.HandleFunc("/services", s.Service().DeleteServiceHandler).Methods(http.MethodDelete)

	adminRouter.HandleFunc("/plans", s.Service().CreatePlanHandler).Methods(http.MethodPost)

	authenticatedRouter.PathPrefix("/").Handler(s.Proxy().ProxyHandler(rootHandler(db)))

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

		// userInt := coremiddleware.User(r.Context())
		// if userInt == nil {
		// 	if err := json.NewEncoder(w).Encode(struct {
		// 		Services []models.Service `json:"services"`
		// 	}{
		// 		Services: servicesList,
		// 	}); err != nil {
		// 		log.Ctx(r.Context()).Err(err).Send()
		// 		http.Error(w, "error marshaling", http.StatusInternalServerError)
		// 		return
		// 	}
		// 	return
		// }

		// user := models.NewUserFromInt(userInt)

		if err := json.NewEncoder(w).Encode(struct {
			User     models.User      `json:"user"`
			Services []models.Service `json:"services"`
		}{
			//User:     user,
			Services: servicesList,
		}); err != nil {
			log.Ctx(r.Context()).Err(err).Send()
			http.Error(w, "error marshaling", http.StatusInternalServerError)
			return
		}
	}
}
