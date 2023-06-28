package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (d Database) CreateUser(ctx context.Context, u models.User) (models.User, error) {
	query := `
		INSERT INTO "user" (id, external_id, email, password,  avatar, firstname, lastname, role, stripe_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT DO NOTHING
		RETURNING id, external_id, email, avatar, firstname, lastname, role, stripe_key, created_at`

	err := d.db.QueryRow(
		ctx,
		query,
		u.ID,
		u.ExternalID,
		u.Email,
		u.Password,
		u.AvatarURL,
		u.Firstname,
		u.Lastname,
		u.Role,
		u.StripeKey,
		u.CreatedAt,
	).Scan(
		&u.ID, &u.ExternalID, &u.Email, &u.AvatarURL, &u.Firstname, &u.Lastname, &u.Role, &u.StripeKey, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return u, nil
		}
		return models.User{}, fmt.Errorf("failed to create user: %w", err)
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
		return models.User{}, fmt.Errorf("failed to update user: %w", err)
	}

	return u, nil
}

func (d *Database) GetUserByID(ctx context.Context, userID uuid.UUID) (models.User, error) {
	query := `
		SELECT id, external_id, email, firstname, lastname, role, stripe_key, created_at, updated_at, deleted_at
		FROM "user"
		WHERE id = $1 AND deleted_at IS NULL
	`

	var user models.User
	err := d.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.ExternalID,
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

func (d *Database) GetFullUserByEmail(ctx context.Context, userEmail string) (models.User, error) {
	query := `
		SELECT id, external_id, email, firstname, lastname, role, stripe_key, created_at, updated_at, deleted_at
		FROM "user"
		WHERE email = $1 AND deleted_at IS NULL
	`

	var user models.User
	err := d.db.QueryRow(ctx, query, userEmail).Scan(
		&user.ID,
		&user.ExternalID,
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
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, nil
		}
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
		return nil, fmt.Errorf("failed to retrieve user services: %w", err)
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var service models.Service
		if err := rows.Scan(
			&service.ID, &service.Name, &service.Prefix, &service.RequiredRoles,
			&service.CreatedAt, &service.UpdatedAt, &service.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user service: %w", err)
		}
		services = append(services, service)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over user services: %w", err)
	}

	return services, nil
}

var (
	ErrUserNotFound = errors.New("user not found")
)

func (d *Database) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User

	err := d.db.QueryRow(ctx, `SELECT id, email, password, role FROM "user" WHERE email = $1 AND deleted_at IS NULL`, email).
		Scan(&user.ID, &user.Email, &user.Password, &user.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, ErrUserNotFound
		}
		return user, err
	}

	return user, nil
}
