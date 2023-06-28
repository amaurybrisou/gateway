package gwservice

import (
	"encoding/json"
	"errors"
	"net/http"
	"text/template"
	"time"

	"github.com/amaurybrisou/gateway/pkg/core/cryptlib"
	"github.com/amaurybrisou/gateway/pkg/core/jwtlib"
	coremiddleware "github.com/amaurybrisou/gateway/pkg/http/middleware"
	"github.com/amaurybrisou/gateway/src/database"
	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type Service struct {
	db  *database.Database
	jwt *jwtlib.JWT
}

func New(db *database.Database, jwt *jwtlib.JWT) Service {
	return Service{
		db:  db,
		jwt: jwt,
	}
}

func (s Service) CreateServiceHandler(w http.ResponseWriter, r *http.Request) {
	var service models.Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	service.ID = uuid.New()

	createdService, err := s.db.CreateService(r.Context(), service)
	if err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(createdService); err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s Service) DeleteServiceHandler(w http.ResponseWriter, r *http.Request) {
	serviceID := r.URL.Query().Get("serviceID")
	if serviceID == "" {
		log.Ctx(r.Context()).Err(errors.New("serviceID missing")).Send()
		http.Error(w, "serviceID parameter is missing", http.StatusBadRequest)
		return
	}

	// Parse the serviceID string into a UUID
	uuidServiceID, err := uuid.Parse(serviceID)
	if err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, "invalid serviceID", http.StatusBadRequest)
		return
	}

	deleted, err := s.db.DeleteService(r.Context(), uuidServiceID)
	if err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		Deleted bool `json:"deleted"`
	}{
		Deleted: deleted,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s Service) GetAllServicesHandler(w http.ResponseWriter, r *http.Request) {
	services, err := s.db.GetServices(r.Context())
	if err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(services); err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s Service) ServicePricePage(w http.ResponseWriter, r *http.Request) {
	serviceName := mux.Vars(r)["service_name"]

	service, err := s.db.GetServiceByName(r.Context(), serviceName)
	if err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>{{.service.Name}}</title>
		</head>
		<body>		
			<script async src="https://js.stripe.com/v3/pricing-table.js"></script>
			<stripe-pricing-table pricing-table-id="{{.service.PricingTableKey}}"
				publishable-key="{{.service.PricingTablePublishableKey}}"
				client-reference-id="{{.service.ID}}"				
				>
			</stripe-pricing-table>
		</body>
		</html>
	`

	// Create a template instance
	t, err := template.New("services").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user := coremiddleware.User(r.Context())

	// Execute the template with the list of services
	w.Header().Set("Content-Type", "text/html")
	err = t.Execute(w, map[any]any{
		"service": service,
		"user":    user,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s Service) LoginHandler(w http.ResponseWriter, r *http.Request) {
	type Credentials struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	// Parse the request body into a Credentials struct
	var creds Credentials
	err := json.NewDecoder(r.Body).Decode(&creds)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("Invalid request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := s.db.GetUserByEmail(r.Context(), creds.Email)
	if err != nil && !errors.Is(err, database.ErrUserNotFound) {
		log.Ctx(r.Context()).Error().Err(err).Msg("internal error")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if user.ID == uuid.Nil || !cryptlib.ValidateHash(creds.Password, user.Password) {
		log.Ctx(r.Context()).Error().Err(err).Msg("invalid credentials")
		http.Error(w, "invalid credentials", http.StatusForbidden)
		return
	}

	// Generate a JWT token with a subject and expiration time
	token, err := s.jwt.GenerateToken(user.ID.String(), time.Now().Add(time.Hour), time.Now())
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("failed to generate")
		http.Error(w, "failed to generate token", http.StatusInternalServerError)
		return
	}

	// Return the token as the response
	response := map[string]string{"token": token}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response) //nolint
}