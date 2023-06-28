package proxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	coremiddleware "github.com/amaurybrisou/gateway/pkg/http/middleware"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type Proxy struct {
	db          *database.Database
	servicesMap map[string]*models.Service
}

func New(db *database.Database) Proxy {
	services, err := db.GetServices(context.Background())
	if err != nil {
		panic(err)
	}

	sI := Proxy{db: db, servicesMap: map[string]*models.Service{}}
	for i := range services {
		s := services[i]
		sI.servicesMap[s.Prefix] = &s
		sI.servicesMap[s.Domain] = &s
	}

	return sI
}

func (s Proxy) ProxyHandler(next http.HandlerFunc) http.HandlerFunc {
	// Create a reverse proxy
	return func(w http.ResponseWriter, r *http.Request) {
		proxy := &httputil.ReverseProxy{
			Director: func(r *http.Request) {
				// Extract the path prefix from the request URL
				pathPrefix := extractPathPrefix(r.URL.Path)

				// Lookup the backend URL based on the path prefix
				service, ok := s.servicesMap[pathPrefix]
				if !ok {
					service, ok = s.servicesMap[r.Host]
					if !ok {
						// No matching backend URL found, return a 404 response
						log.Ctx(r.Context()).Error().Msg("backend not found")
						next(w, r)
						return
					}
				}

				userID, ok := r.Context().Value(coremiddleware.UserIDCtxKey).(uuid.UUID)
				if !ok {
					// log.Ctx(r.Context()).Err(errors.New("invalid user_id")).Send()
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				hasRole, err := s.db.HasRole(r.Context(), userID, service.RequiredRoles...)
				if err != nil {
					log.Ctx(r.Context()).Err(err).Msg("determine user roles")
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				if !hasRole {
					next(w, r)
					return
				}

				// Set the backend URL as the target URL for the reverse proxy
				targetURL, err := url.Parse(service.Host)
				if err != nil {
					log.Ctx(r.Context()).Err(err).Msg("Failed to parse backend URL")
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				// Set the target URL for the reverse proxy
				r.URL.Scheme = targetURL.Scheme
				r.URL.Host = targetURL.Host
				r.URL.Path = singleJoiningSlash(targetURL.Path, r.URL.Path)
				r.Host = targetURL.Host
			},
		}

		services, err := s.db.GetServices(context.Background())
		if err != nil {
			log.Ctx(r.Context()).Err(err).Msg("fetch services")
			http.Error(w, "fetch services", http.StatusInternalServerError)
			return
		}

		s.servicesMap = make(map[string]*models.Service, len(services)*2)
		for i := range services {
			cs := services[i]
			s.servicesMap[cs.Prefix] = &cs
			s.servicesMap[cs.Domain] = &cs
		}

		proxy.ServeHTTP(w, r)
	}
}

// Helper function to extract the path prefix from a URL.
func extractPathPrefix(path string) string {
	path = strings.Split(path, "/")[1] // Assuming the path has a leading slash
	return "/" + strings.Split(path, "/")[0]
}

// Helper function to join URL paths with a single slash.
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
