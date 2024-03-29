package models

import (
	"time"

	ablibmodels "github.com/amaurybrisou/ablib/models"
	"github.com/google/uuid"
)

type UserRole struct {
	UserID         uuid.UUID         `json:"user"`
	SubscriptionID string            `json:"subscription_id"`
	Role           Role              `json:"role"`
	Metadata       map[string]string `json:"metadata"`
	ExpiresAt      *time.Time        `json:"expires_at"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      *time.Time        `json:"updated_at"`
	DeletedAt      *time.Time        `json:"deleted_at"`
}

type Role string

const (
	EmptyRole Role = "{}"
	TempRole  Role = "{'temporary-role'}"

	ServiceStatusOK = "OK"
)

type Service struct {
	ID                         uuid.UUID `json:"id"`
	Name                       string    `json:"name"`
	Description                string    `json:"description"`
	Prefix                     string    `json:"prefix"`
	Domain                     string    `json:"domain"`
	Host                       string    `json:"host"`
	ImageURL                   *string   `json:"image_url"`
	PricingTableKey            string    `json:"pricing_table_key"`
	PricingTablePublishableKey string    `json:"pricing_table_publishable_key"`

	RetryCount int `json:"-"`

	RequiredRoles []Role     `json:"required_roles"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at"`
	HasAccess     *bool      `json:"has_access"`
	IsFree        *bool      `json:"is_free"`
}

func (s Service) GetHost() string {
	return s.Host
}

func (s Service) GetRetryCount() int {
	return s.RetryCount
}

func (s *Service) SetRetryCount(c int) {
	s.RetryCount = c
}

func (s Service) GetID() uuid.UUID {
	return s.ID
}

func (s *Service) SetStatus(st string) {
	s.Status = st
}

func (s Service) GetStatus() string {
	return string(s.Status)
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
	ID         uuid.UUID               `json:"id"`
	ExternalID string                  `json:"external_id"`
	Email      string                  `json:"email"`
	AvatarURL  string                  `json:"avatar"`
	Firstname  string                  `json:"firstname"`
	Lastname   string                  `json:"lastname"`
	Password   string                  `json:"-"`
	Role       ablibmodels.GatewayRole `json:"role"`
	StripeKey  *string                 `json:"-"`
	IsNew      string                  `json:"-"`
	CreatedAt  time.Time               `json:"created_at"`
	UpdatedAt  *time.Time              `json:"updated_at"`
	DeletedAt  *time.Time              `json:"deleted_at"`
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

func (u User) GetPassword() string {
	return u.Password
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

func (u User) GetRole() ablibmodels.GatewayRole {
	return u.Role
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

func NewUserFromInt(u ablibmodels.UserInterface) User {
	return User{
		ID:         u.GetID(),
		ExternalID: u.GetExternalID(),
		Email:      u.GetEmail(),
		AvatarURL:  u.GetAvatarURL(),
		Firstname:  u.GetFirstname(),
		Lastname:   u.GetLastname(),
		Role:       u.GetRole(),
		CreatedAt:  u.GetCreatedAt(),
		UpdatedAt:  u.GetUpdatedAt(),
		DeletedAt:  u.GetDeletedAt(),
	}
}
