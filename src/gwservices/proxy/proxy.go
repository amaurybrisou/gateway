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

	// Lookup the backend URL based on the path prefix
	service, err := s.db.GetServiceByPrefixOrDomain(r.Context(), pathPrefix, r.Host)
	if err != nil {
		log.Ctx(r.Context()).Warn().Err(err).Msg("backend not found")
		http.Redirect(w, r, s.notFoundRedirectURL, http.StatusPermanentRedirect)
		return
	}

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

	// Set the backend URL as the target URL for the reverse proxy
	targetURL, err := url.Parse(service.Host)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("Failed to parse backend URL")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	trim, err := url.JoinPath(s.stripPrefix, pathPrefix)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("Failed to parse backend URL")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = targetURL.Scheme
			pr.Out.URL.Host = targetURL.Host
			pr.Out.URL.Path = strings.TrimPrefix(pr.Out.URL.Path, trim)
			pr.Out.Header.Add("X-Request-Id", middleware.GetReqID(pr.In.Context()))
			pr.Out.Header.Add("X-Forwarded-For", pr.In.RemoteAddr)
			pr.Out.Host = targetURL.Host
		},
	}

	proxy.ServeHTTP(w, r)
}

// Helper function to extract the path prefix from a URL.
func (p Proxy) extractPathPrefix(path string) string {
	path = strings.Split(strings.TrimPrefix(path, p.stripPrefix), "/")[1] // Assuming the path has a leading slash
	return "/" + strings.Split(path, "/")[0]
}
