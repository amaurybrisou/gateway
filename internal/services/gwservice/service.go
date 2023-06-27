package gwservice

import (
	"encoding/json"
	"errors"
	"net/http"
	"text/template"

	"github.com/amaurybrisou/gateway/internal/db"
	"github.com/amaurybrisou/gateway/internal/db/models"
	coremiddleware "github.com/amaurybrisou/gateway/pkg/http/middleware"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

type Service struct {
	db *db.Database
}

func New(db *db.Database) Service {
	return Service{
		db: db,
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

func (s Service) CreatePlanHandler(w http.ResponseWriter, r *http.Request) {
	var plan models.Plan
	if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	plan.ID = uuid.New()

	createdPlan, err := s.db.CreatePlan(r.Context(), plan.ServiceID, plan.Name, plan.Description, plan.Price, plan.Duration, plan.Currency)
	if err != nil {
		log.Ctx(r.Context()).Err(err).Send()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(createdPlan); err != nil {
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
