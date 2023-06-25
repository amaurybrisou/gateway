package coremodels

import (
	"time"

	"github.com/google/uuid"
)

type GatewayRole string

const (
	ADMIN GatewayRole = "ADMIN"
	USER  GatewayRole = "USER"
)

type UserInterface interface {
	GetID() uuid.UUID
	GetEmail() string
	GetAvatarURL() string
	GetFirstname() string
	GetLastname() string
	GetRole() GatewayRole
	GetStripeKey() *string
	GetCreatedAt() time.Time
	GetUpdatedAt() *time.Time
	GetDeletedAt() *time.Time
}

type User struct {
	ID        uuid.UUID
	Email     string
	AvatarURL string
	Firstname string
	Lastname  string
	Role      GatewayRole
	StripeKey *string
	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
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

func (u User) GetRole() GatewayRole {
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
