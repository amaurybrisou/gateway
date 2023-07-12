package src

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/amaurybrisou/ablib"
	ablibhttp "github.com/amaurybrisou/ablib/http"
	ablibmodels "github.com/amaurybrisou/ablib/models"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/gwservices"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

func Router(s gwservices.Services, db *database.Database) http.Handler {
	r := chi.NewRouter()

	if ablib.LookupEnv("ENV", "dev") == "dev" {
		r.Use(cors.AllowAll().Handler)
	}

	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(ablibhttp.LoggerMiddleware(&log.Logger))

	r.Use(ablibhttp.NewRateLimitMiddleware(ablibhttp.WithRateLimit(
		rate.Limit(ablib.LookupEnvFloat64("RATE_LIMIT", float64(5))),
		ablib.LookupEnvInt("RATE_LIMIT_BURST", 10),
	)).Middleware)
	r.Use(ablibhttp.RequestMetric("gateway"))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(time.Second * 10))

	// jwtAuthProvider := ablibhttp.NewJwtAuth(
	// 	s.Jwt(),
	// 	func(ctx context.Context, email string) (ablibmodels.UserInterface, error) {
	// 		return db.GetUserByEmail(ctx, email)
	// 	},
	// 	func(ctx context.Context, id uuid.UUID) (ablibmodels.UserInterface, error) {
	// 		return db.GetUserByID(ctx, id)
	// 	},
	// )

	authProvider := ablibhttp.NewCookieAuthHandler(
		ablib.LookupEnv("COOKIE_SCRET", "something-secret"),
		ablib.LookupEnv("COOKIE_NAME", "cookie-name"),
		ablib.LookupEnvInt("COOKIE_MAX_AGE", 3600),
		func(ctx context.Context, email string) (ablibmodels.UserInterface, error) {
			return db.GetUserByEmail(ctx, email)
		},
		func(ctx context.Context, id uuid.UUID) (ablibmodels.UserInterface, error) {
			return db.GetUserByID(ctx, id)
		},
		s.Jwt(),
	)

	// UNAUTHENTICATED
	r.Handle("/", http.RedirectHandler("/home", http.StatusPermanentRedirect))
	r.Route("/home", func(r chi.Router) {
		r.Handle("/*", http.StripPrefix("/home", http.FileServer(http.Dir(ablib.LookupEnv("FRONT_BUILD_PATH", "front/dist")))))
	})
	r.Post("/login", authProvider.Login)

	r.Post("/payment/webhook", s.Payment().StripeWebhook)
	r.Get("/services", s.Service().GetAllServicesHandler)
	r.Get("/pricing/{service_name}", s.Service().ServicePricePage)

	// AUTHENTICATED

	r.Route("/auth", func(authenticatedRouter chi.Router) {
		authenticatedRouter.Use(authProvider.Middleware)
		authenticatedRouter.Use(ablibhttp.JsonContentType())

		authenticatedRouter.Post("/update-password", s.Service().PasswordUpdateHandler)
		authenticatedRouter.Get("/user", s.Service().GetUserHandler)
		authenticatedRouter.Get("/logout", authProvider.Logout)

		authenticatedRouter.Get("/services", s.Service().GetAllServicesHandler)

		authenticatedRouter.Route("/admin", func(adminRouter chi.Router) {
			adminRouter.Use(ablibhttp.IsAdminMiddleware)

			adminRouter.Post("/services", s.Service().CreateServiceHandler)
			adminRouter.Delete("/services/{service_id}", s.Service().DeleteServiceHandler)
			adminRouter.Get("/services", s.Service().GetAllServicesHandler)
			adminRouter.Get("/version", Version)
		})
	})

	r.Route("/{service_name}", func(r chi.Router) {
		r.HandleFunc("/*", s.Proxy().ServiceAccessHandler(authProvider.Middleware))
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
