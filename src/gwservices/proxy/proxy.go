package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	ablibhttp "github.com/amaurybrisou/ablib/http"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Proxy struct {
	db                  *database.Database
	stripPrefix         string
	notFoundRedirectURL string
	noRoleRedirectURL   string
}

type Config struct {
	StripPrefix         string
	NotFoundRedirectURL string
	NoRoleRedirectURL   string
}

func New(db *database.Database, cfg Config) Proxy {
	return Proxy{
		db:                  db,
		stripPrefix:         cfg.StripPrefix,
		notFoundRedirectURL: cfg.NotFoundRedirectURL,
		noRoleRedirectURL:   cfg.NoRoleRedirectURL,
	}
}

func (s Proxy) PublicRoutes(w http.ResponseWriter, r *http.Request) {
	pathPrefix := chi.URLParam(r, "service_name")
	if pathPrefix == "" {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}

	// pathPrefix := p.extractPathPrefix(r.URL.Path)
	log.Ctx(r.Context()).Debug().Any("prefix", pathPrefix).Any("url.path", r.URL.Path).Msg("proxy request received")

	// Lookup the backend URL based on the path prefix
	service, err := s.db.GetServiceByName(r.Context(), pathPrefix)
	if err != nil {
		log.Ctx(r.Context()).Warn().Err(err).Msg("backend not found")
		http.Redirect(w, r, s.notFoundRedirectURL, http.StatusPermanentRedirect)
		return
	}

	s.ProxyHandler(service, w, r).ServeHTTP(w, r)
}

func (s Proxy) ProxyHandler(service models.Service, w http.ResponseWriter, r *http.Request) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetURL, err := url.Parse(service.Host)
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("Failed to parse backend URL")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Ctx(r.Context()).Debug().Any("domain", service.Domain).Any("host", r.Host).Send()
		if service.Domain != r.Host && service.Domain != "" {
			http.Redirect(w, r, fmt.Sprintf("https://%s%s", service.Domain, strings.TrimPrefix(r.URL.Path, service.Prefix)), http.StatusPermanentRedirect)
			return
		}

		proxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = targetURL.Scheme
				req.URL.Host = targetURL.Host
				req.URL.Path = strings.TrimPrefix(req.URL.Path, service.Prefix)
				req.Header.Add("X-Request-Id", middleware.GetReqID(req.Context()))
				req.Header.Add("X-Forwarded-For", req.RemoteAddr)
				req.Host = targetURL.Host
				log.Ctx(r.Context()).Debug().Any("path", req.URL.Path).Any("host", req.Host).Msg("proxying to")
			},
		}

		proxy.ServeHTTP(w, r)
	})
}

func (p Proxy) ServiceAccessHandler(authMiddleware func(next http.Handler) http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pathPrefix := "/" + chi.URLParam(r, "service_name")

		log.Ctx(r.Context()).Debug().
			Any("host", r.Host).
			Any("prefix", pathPrefix).
			Any("url.path", r.URL.Path).Msg("proxy request received")
		// Lookup the backend URL based on the path prefix
		service, err := p.db.GetServiceByPrefixOrDomain(r.Context(), pathPrefix, r.Host)
		if err != nil {
			log.Ctx(r.Context()).Warn().Err(err).Msg("backend not found")
			http.Redirect(w, r, p.notFoundRedirectURL, http.StatusPermanentRedirect)
			return
		}

		if len(service.RequiredRoles) > 0 {
			// Service requires authentication, perform JWT authentication
			authMiddleware(p.CheckRequiredRoles(service, p.ProxyHandler(service, w, r))).ServeHTTP(w, r)
		} else {
			// Service does not require authentication, continue to the next handler
			p.ProxyHandler(service, w, r).ServeHTTP(w, r)
		}
	}
}

func (p Proxy) CheckRequiredRoles(service models.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(ablibhttp.UserIDCtxKey).(uuid.UUID)
		if !ok {
			log.Ctx(r.Context()).Error().Err(errors.New("invalid user_id")).Send()
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		userRole, err := p.db.GetUserRole(r.Context(), userID, service.RequiredRoles[0])
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("determine user roles")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if userRole.UserID == uuid.Nil {
			http.Redirect(w, r, p.noRoleRedirectURL+"/"+service.Name, http.StatusTemporaryRedirect)
			return
		}

		m, err := json.Marshal(userRole.Metadata)
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("marshal metadata")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		r.Header.Add("X-Plan-Metadata", string(m))

		user := ablibhttp.User(r.Context())
		r.Header.Add("X-Stripe-Customer-ID", user.GetExternalID())

		next.ServeHTTP(w, r)
	})
}

// func (p Proxy) extractPathPrefix(path string) string {
// 	path = strings.TrimPrefix(path, p.stripPrefix)
// 	parts := strings.Split(path, "/")
// 	if len(parts) > 1 {
// 		return "/" + parts[1]
// 	}
// 	return path
// }
