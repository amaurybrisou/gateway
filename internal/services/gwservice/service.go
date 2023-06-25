package gwservice

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/amaurybrisou/gateway/internal/db"
	"github.com/amaurybrisou/gateway/internal/db/models"
	"github.com/google/uuid"
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
