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
	"github.com/amaurybrisou/gateway/src/serializer"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
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

	if err := json.NewEncoder(w).Encode(serializer.Service(&createdService)); err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s Service) DeleteServiceHandler(w http.ResponseWriter, r *http.Request) {
	serviceID := chi.URLParam(r, "service_id")
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

	if err := json.NewEncoder(w).Encode(serializer.Services(services)); err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s Service) ServicePricePage(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "service_name")
	if serviceName == "" {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}

	service, err := s.db.GetServiceByName(r.Context(), serviceName)
	if err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if service.ID == uuid.Nil {
		http.Error(w, "service not found", http.StatusNotFound)
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
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
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

// PasswordUpdateHandler is an HTTP handler for updating the user password.
func (s Service) PasswordUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("Invalid request body")
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	cipheredPassword, err := cryptlib.GenerateHash(request.Password, bcrypt.DefaultCost)
	if err != nil {
		log.Ctx(r.Context()).Error().Err(err).Msg("cipher password")
		http.Error(w, "could not cipher password", http.StatusInternalServerError)
		return
	}

	user, err := s.db.UpdatePassword(r.Context(), request.Email, cipheredPassword)
	if err != nil {
		if errors.Is(err, database.ErrUserNotFound) {
			log.Ctx(r.Context()).Error().Err(err).Msg("user not found")
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			log.Ctx(r.Context()).Error().Err(err).Msg("update password")
			http.Error(w, "Failed to update password", http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(user) //nolint
}
