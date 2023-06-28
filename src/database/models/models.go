package models

import (
	"time"

	coremodels "github.com/amaurybrisou/gateway/pkg/core/models"
	"github.com/google/uuid"
)

type UserRole struct {
	UserID         uuid.UUID  `json:"user"`
	SubscriptionID string     `json:"subscription_id"`
	Role           Role       `json:"role"`
	ExpiresAt      *time.Time `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at"`
}

type Role string

const (
	EmptyRole Role = "{}"
)

type Service struct {
	ID                         uuid.UUID `json:"id"`
	Name                       string    `json:"name"`
	Prefix                     string    `json:"prefix"`
	Domain                     string    `json:"domain"`
	Host                       string    `json:"host"`
	ImageURL                   *string   `json:"image_url"`
	PricingTableKey            string    `json:"pricing_table_key"`
	PricingTablePublishableKey string    `json:"pricing_table_publishable_key"`

	RequiredRoles []Role     `json:"required_roles"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
}

func (s Service) HasRole(r Role) bool {
	for _, cr := range s.RequiredRoles {
		if r == cr {
			return true
		}
	}
	return false
}

type User struct {
	ID         uuid.UUID              `json:"id"`
	ExternalID string                 `json:"external_id"`
	Email      string                 `json:"email"`
	AvatarURL  string                 `json:"avatar"`
	Firstname  string                 `json:"firstname"`
	Lastname   string                 `json:"lastname"`
	Password   string                 `json:"-"`
	Role       coremodels.GatewayRole `json:"role"`
	StripeKey  *string                `json:"-"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  *time.Time             `json:"updated_at"`
	DeletedAt  *time.Time             `json:"deleted_at"`
}

func (u User) GetID() uuid.UUID {
	return u.ID
}

func (u User) GetExternalID() string {
	return u.ExternalID
}

func (u User) GetEmail() string {
	return u.Email
}

func (u User) GetAvatarURL() string {
	return u.AvatarURL
}

func (u User) GetFirstname() string {
	return u.Firstname
}

func (u User) GetLastname() string {
	return u.Lastname
}

func (u User) GetRole() coremodels.GatewayRole {
	return u.Role
}

func (u User) GetStripeKey() *string {
	return u.StripeKey
}

func (u User) GetCreatedAt() time.Time {
	return u.CreatedAt
}

func (u User) GetUpdatedAt() *time.Time {
	return u.UpdatedAt
}

func (u User) GetDeletedAt() *time.Time {
	return u.DeletedAt
}

func NewUserFromInt(u coremodels.UserInterface) User {
	return User{
		ID:         u.GetID(),
		ExternalID: u.GetExternalID(),
		Email:      u.GetEmail(),
		AvatarURL:  u.GetAvatarURL(),
		Firstname:  u.GetFirstname(),
		Lastname:   u.GetLastname(),
		Role:       u.GetRole(),
		StripeKey:  u.GetStripeKey(),
		CreatedAt:  u.GetCreatedAt(),
		UpdatedAt:  u.GetUpdatedAt(),
		DeletedAt:  u.GetDeletedAt(),
	}
}
