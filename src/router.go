package src

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	coremiddleware "github.com/amaurybrisou/ablib/http/middleware"
	coremodels "github.com/amaurybrisou/ablib/models"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/gwservices"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

func Router(s gwservices.Services, db *database.Database, publicRootPath string, rateLimit float64, burst int) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(coremiddleware.LoggerMiddleware(&log.Logger))
	r.Use(coremiddleware.NewRateLimitMiddleware(coremiddleware.WithRateLimit(rate.Limit(rateLimit), burst)).Middleware)
	r.Use(coremiddleware.RequestMetric("gateway"))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(time.Second * 10))

	// UNAUTHENTICATED
	r.Handle("/", http.RedirectHandler("/home", http.StatusPermanentRedirect))
	r.Handle("/home/*", http.FileServer(http.Dir(publicRootPath)))
	r.Post("/login", s.Service().PostLoginHandler)

	r.Post("/payment/webhook", s.Payment().StripeWebhook)
	r.With(coremiddleware.JsonContentType()).Get("/services", s.Service().GetAllServicesHandler)
	r.Get("/pricing/{service_name}", s.Service().ServicePricePage)

	// AUTHENTICATED
	authMiddleware := coremiddleware.NewAuthMiddleware(s.Jwt(), func(ctx context.Context, id uuid.UUID) (coremodels.UserInterface, error) {
		return db.GetUserByID(ctx, id)
	})

	r.Route("/auth", func(authenticatedRouter chi.Router) {
		authenticatedRouter.Use(authMiddleware.JWTAuth)
		authenticatedRouter.Use(coremiddleware.JsonContentType())

		authenticatedRouter.Post("/update-password", s.Service().PasswordUpdateHandler)
		authenticatedRouter.Get("/user", s.Service().GetUserHandler)
		authenticatedRouter.Get("/services", s.Service().GetAllServicesHandler)

		authenticatedRouter.Route("/admin", func(adminRouter chi.Router) {
			adminRouter.Use(authMiddleware.IsAdmin)

			adminRouter.Post("/services", s.Service().CreateServiceHandler)
			adminRouter.Delete("/services/{service_id}", s.Service().DeleteServiceHandler)
			adminRouter.Get("/services", s.Service().GetAllServicesHandler)
			adminRouter.Get("/version", Version)
		})
	})

	r.HandleFunc("/*", s.Proxy().ServiceAccessHandler(authMiddleware.JWTAuth))

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
