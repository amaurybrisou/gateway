package proxy

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	coremiddleware "github.com/amaurybrisou/gateway/pkg/http/middleware"
	"github.com/amaurybrisou/gateway/src/database"
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

func (s Proxy) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	pathPrefix := s.extractPathPrefix(r.URL.Path)
	log.Ctx(r.Context()).Debug().Any("prefix", pathPrefix).Any("url.path", r.URL.Path).Msg("proxy request received")

	// Lookup the backend URL based on the path prefix
	service, err := s.db.GetServiceByPrefixOrDomain(r.Context(), pathPrefix, r.Host)
	if err != nil {
		log.Ctx(r.Context()).Warn().Err(err).Msg("backend not found")
		http.Redirect(w, r, s.notFoundRedirectURL, http.StatusPermanentRedirect)
		return
	}

	if len(service.RequiredRoles) > 0 {
		userID, ok := r.Context().Value(coremiddleware.UserIDCtxKey).(uuid.UUID)
		if !ok {
			log.Ctx(r.Context()).Error().Err(errors.New("invalid user_id")).Send()
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		hasRole, err := s.db.HasRole(r.Context(), userID, service.RequiredRoles...)
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("determine user roles")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !hasRole {
			http.Redirect(w, r, s.noRoleRedirectURL+"/"+service.Name, http.StatusTemporaryRedirect)
			return
		}
	}

	// Set the backend URL as the target URL for the reverse proxy
	targetURL, err := url.Parse(service.Host)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("Failed to parse backend URL")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path = strings.TrimPrefix(req.URL.Path, pathPrefix)
			req.Header.Add("X-Request-Id", middleware.GetReqID(req.Context()))
			req.Header.Add("X-Forwarded-For", req.RemoteAddr)
			req.Host = targetURL.Host
			log.Ctx(r.Context()).Debug().Any("url", r.URL).Any("host", r.Host).Msg("prosying to")
		},
	}

	proxy.ServeHTTP(w, r)
}

func (p Proxy) extractPathPrefix(path string) string {
	path = strings.TrimPrefix(path, p.stripPrefix)
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		return "/" + parts[1]
	}
	return path
}
