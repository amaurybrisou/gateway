package db

import (
	"context"
	"fmt"

	"github.com/amaurybrisou/gateway/internal/db/models"
	coremodels "github.com/amaurybrisou/gateway/pkg/core/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (d Database) CreateUserAndToken(ctx context.Context, user models.User, token models.AccessToken) error {
	tx, err := d.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}

	// Create user
	userQuery := `
		INSERT INTO "user" (id, email, avatar, firstname, lastname, role, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING`

	_, err = tx.Exec(ctx, userQuery, user.ID, user.Email, user.AvatarURL, user.Firstname, user.Lastname, user.Role, user.CreatedAt)
	if err != nil {
		tx.Rollback(ctx) //nolint
		return fmt.Errorf("failed to create user: %v", err)
	}

	// Create access token
	tokenQuery := `
		INSERT INTO access_token (user_id, external_id, token, expires_at)
		VALUES ($1, $2, $3, $4) ON CONFLICT(external_id) 
		DO UPDATE SET token = excluded.token, expires_at = excluded.expires_at`

	_, err = tx.Exec(ctx, tokenQuery, token.UserID, token.ExternalID, token.Token, token.ExpiresAt)
	if err != nil {
		tx.Rollback(ctx) //nolint
		return fmt.Errorf("failed to create access token: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func (d Database) CreateUser(ctx context.Context, u models.User) (models.User, error) {
	query := `
		INSERT INTO "user" (id, email, avatar, firstname, lastname, role, stripe_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING *`

	err := d.db.QueryRow(ctx, query, u.ID, u.Email, u.AvatarURL, u.Firstname, u.Lastname, u.Role, u.StripeKey, u.CreatedAt).Scan(
		&u.ID, &u.Email, &u.AvatarURL, &u.Firstname, &u.Lastname, &u.Role, &u.StripeKey, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to create user: %v", err)
	}

	return u, nil
}

func (d Database) UpdateUser(ctx context.Context, u models.User) (models.User, error) {
	query := `
		UPDATE user
		SET avatar = $1, email = $2, firstname = $3, lastname = $4, 
			role = $5, stripe_key = $6, updated_at = $7
		WHERE id = $8
		RETURNING *`

	err := d.db.QueryRow(ctx, query, u.AvatarURL, u.Email, u.Firstname, u.Lastname, u.Role, u.StripeKey, u.UpdatedAt, u.ID).Scan(
		&u.ID, &u.Email, &u.Firstname, &u.Lastname, &u.Role, &u.StripeKey, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to update user: %v", err)
	}

	return u, nil
}

func (d *Database) GetUserByID(ctx context.Context, userID uuid.UUID) (models.User, error) {
	query := `
		SELECT id, email, firstname, lastname, role, stripe_key, created_at, updated_at, deleted_at
		FROM "user"
		WHERE id = $1 AND deleted_at IS NULL
	`

	var user models.User
	err := d.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Firstname,
		&user.Lastname,
		&user.Role,
		&user.StripeKey,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		// Handle the error (e.g., return an error response, log the error)
		return models.User{}, err
	}

	return user, nil
}

func (d Database) GetUserByAccessToken(ctx context.Context, token string) (coremodels.UserInterface, error) {
	query := `
		SELECT u.id, u.email, u.firstname, u.lastname, u.role, u.stripe_key, u.created_at, u.updated_at
		FROM "user" u
		LEFT JOIN access_token at ON u.id = at.user_id
		WHERE at.token = $1
		AND u.deleted_at IS NULL
	`

	var user models.User
	err := d.db.QueryRow(ctx, query, token).Scan(
		&user.ID,
		&user.Email,
		&user.Firstname,
		&user.Lastname,
		&user.Role,
		&user.StripeKey,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (d Database) GetUserServices(ctx context.Context, userID uuid.UUID) ([]models.Service, error) {
	query := `
		SELECT s.*
		FROM service s
		INNER JOIN user_role ur ON ur.user_id = $1 AND ur.role = ANY(s.required_roles)`

	rows, err := d.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user services: %v", err)
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var service models.Service
		if err := rows.Scan(
			&service.ID, &service.Name, &service.Prefix, &service.RequiredRoles,
			&service.CreatedAt, &service.UpdatedAt, &service.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user service: %v", err)
		}
		services = append(services, service)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over user services: %v", err)
	}

	return services, nil
}
