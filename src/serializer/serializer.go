package serializer

import (
	"time"

	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
)

type PublicService struct {
	ID                         uuid.UUID  `json:"id,omitempty"`
	Name                       string     `json:"name,omitempty"`
	Description                string     `json:"description,omitempty"`
	Prefix                     string     `json:"prefix,omitempty"`
	Domain                     string     `json:"domain,omitempty"`
	Host                       string     `json:"host,omitempty"`
	ImageURL                   *string    `json:"image_url,omitempty"`
	Status                     string     `json:"status,omitempty"`
	PricingTableKey            string     `json:"pricing_table_key,omitempty"`
	PricingTablePublishableKey string     `json:"pricing_table_publishable_key,omitempty"`
	CreatedAt                  time.Time  `json:"created_at,omitempty"`
	UpdatedAt                  *time.Time `json:"updated_at,omitempty"`
	DeletedAt                  *time.Time `json:"deleted_at,omitempty"`
}

type PublicUser struct {
	ID        uuid.UUID  `json:"id,omitempty"`
	Email     string     `json:"email,omitempty"`
	AvatarURL string     `json:"avatar,omitempty"`
	Firstname string     `json:"firstname,omitempty"`
	Lastname  string     `json:"lastname,omitempty"`
	Role      string     `json:"role,omitempty"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func Service(service *models.Service) *PublicService {
	return &PublicService{
		ID:                         service.ID,
		Name:                       service.Name,
		Description:                service.Description,
		Prefix:                     service.Prefix,
		Domain:                     service.Domain,
		Host:                       service.Host,
		ImageURL:                   service.ImageURL,
		PricingTableKey:            service.PricingTableKey,
		PricingTablePublishableKey: service.PricingTablePublishableKey,
		Status:                     service.Status,
		CreatedAt:                  service.CreatedAt,
		UpdatedAt:                  service.UpdatedAt,
		DeletedAt:                  service.DeletedAt,
	}
}

func Services(services []*models.Service) []*PublicService {
	result := make([]*PublicService, len(services))
	for i, service := range services {
		result[i] = Service(service)
	}
	return result
}

func User(user *models.User) *PublicUser {
	return &PublicUser{
		ID:        user.ID,
		Email:     user.Email,
		AvatarURL: user.AvatarURL,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		DeletedAt: user.DeletedAt,
	}
}

func Users(users []*models.User) []*PublicUser {
	result := make([]*PublicUser, len(users))
	for i, user := range users {
		result[i] = User(user)
	}
	return result
}
