package models

import (
	"time"

	coremodels "github.com/amaurybrisou/gateway/pkg/core/models"
	"github.com/google/uuid"
)

type UserRole struct {
	UserID         uuid.UUID  `json:"user"`
	Role           Role       `json:"role"`
	ExpirationTime time.Time  `json:"expiration_time"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at"`
}

type Role string

type Service struct {
	ID            uuid.UUID               `json:"id"`
	Name          string                  `json:"name"`
	Prefix        string                  `json:"prefix"`
	Domain        string                  `json:"domain"`
	Host          string                  `json:"host"`
	RequiredRoles []Role                  `json:"required_roles"`
	Costs         map[SubDuration]float32 `json:"costs"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     *time.Time              `json:"updated_at"`
	DeletedAt     *time.Time              `json:"deleted_at"`
}

func (s Service) HasRole(r Role) bool {
	for _, cr := range s.RequiredRoles {
		if r == cr {
			return true
		}
	}
	return false
}

type SubDuration time.Duration

const (
	Monthly SubDuration = SubDuration(time.Hour * 24 * 30)
	// ...
	Yearly SubDuration = SubDuration(time.Hour * 24 * 365)
)

type User struct {
	ID        uuid.UUID              `json:"id"`
	Email     string                 `json:"email"`
	AvatarURL string                 `json:"avatar"`
	Firstname string                 `json:"firstname"`
	Lastname  string                 `json:"lastname"`
	Role      coremodels.GatewayRole `json:"role"`
	StripeKey *string                `json:"-"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt *time.Time             `json:"updated_at"`
	DeletedAt *time.Time             `json:"deleted_at"`
}

func (u User) GetID() uuid.UUID {
	return u.ID
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
		ID:        u.GetID(),
		Email:     u.GetEmail(),
		AvatarURL: u.GetAvatarURL(),
		Firstname: u.GetFirstname(),
		Lastname:  u.GetLastname(),
		Role:      u.GetRole(),
		StripeKey: u.GetStripeKey(),
		CreatedAt: u.GetCreatedAt(),
		UpdatedAt: u.GetUpdatedAt(),
		DeletedAt: u.GetDeletedAt(),
	}
}

type AccessToken struct {
	UserID     uuid.UUID `json:"user_id"`
	ExternalID string    `json:"-"`
	Token      string    `json:"token"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type PaymentStatus string

const (
	PaymentPending  PaymentStatus = "PENDING"
	PaymentPaid     PaymentStatus = "PAID"
	PaymentRejected PaymentStatus = "REJECTED"
)

type UserPayment struct {
	ID        uuid.UUID     `json:"id"`
	ServiceID uuid.UUID     `json:"service_id"`
	CreatedAt time.Time     `json:"created_at"`
	Status    PaymentStatus `json:"status"`
	UpdatedAt *time.Time    `json:"updated_at"`
	DeletedAt *time.Time    `json:"deleted_at"`
}
