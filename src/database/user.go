package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	userSelectFields     = "id, external_id, email, avatar, firstname, lastname, role, stripe_key, created_at"
	userSelectFieldsFull = "id, external_id, email, avatar, firstname, lastname, role, stripe_key, created_at, updated_at, deleted_at"
)

func (d Database) CreateUser(ctx context.Context, u models.User) (models.User, error) {
	query := `
		INSERT INTO "user" (` + userSelectFields + `, password)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT DO NOTHING
		RETURNING ` + userSelectFields

	err := d.db.QueryRow(
		ctx,
		query,
		u.ID,
		u.ExternalID,
		u.Email,
		u.AvatarURL,
		u.Firstname,
		u.Lastname,
		u.Role,
		u.StripeKey,
		time.Now(),
		u.Password,
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
		SELECT ` + userSelectFieldsFull + `
		FROM "user"
		WHERE id = $1 AND deleted_at IS NULL
	`

	row := d.db.QueryRow(ctx, query, userID)
	user, err := scanUserFull(row)
	if err != nil {
		// Handle the error (e.g., return an error response, log the error)
		return models.User{}, err
	}

	return user, nil
}

func (d *Database) GetFullUserByEmail(ctx context.Context, userEmail string) (models.User, error) {
	query := `
		SELECT ` + userSelectFieldsFull + `
		FROM "user"
		WHERE email = $1 AND deleted_at IS NULL
	`

	row := d.db.QueryRow(ctx, query, userEmail)
	user, err := scanUserFull(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, nil
		}
		return models.User{}, err
	}

	return user, nil
}

func (d *Database) GetFullUserByExternalID(ctx context.Context, externalID string) (models.User, error) {
	query := `
		SELECT ` + userSelectFieldsFull + `
		FROM "user"
		WHERE external_id = $1 AND deleted_at IS NULL
	`

	row := d.db.QueryRow(ctx, query, externalID)
	user, err := scanUserFull(row)
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
	row := d.db.QueryRow(ctx, `SELECT `+userSelectFields+`, password FROM "user" WHERE email = $1 AND deleted_at IS NULL`, email)
	user, err := scanUserWithPassword(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, ErrUserNotFound
		}
		return user, err
	}

	return user, nil
}

// UpdatePassword set the password and is_new to false, update_at = now and return the user.
func (d *Database) UpdatePassword(ctx context.Context, email, password string) (models.User, error) {
	var user models.User

	row := d.db.QueryRow(ctx, `
		UPDATE "user" SET password = $2, is_new = false, updated_at = now()
		WHERE email = $1 AND deleted_at IS NULL
		RETURNING `+userSelectFieldsFull, email, password)
	user, err := scanUserFull(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, ErrUserNotFound
		}
		return user, err
	}

	return user, nil
}

func scanUserWithPassword(row localRow) (u models.User, err error) {
	err = row.Scan(
		&u.ID,
		&u.ExternalID,
		&u.Email,
		&u.AvatarURL,
		&u.Firstname,
		&u.Lastname,
		&u.Role,
		&u.StripeKey,
		&u.CreatedAt,
		&u.Password,
	)
	return u, err
}

func scanUserFull(row localRow) (u models.User, err error) {
	err = row.Scan(
		&u.ID,
		&u.ExternalID,
		&u.Email,
		&u.AvatarURL,
		&u.Firstname,
		&u.Lastname,
		&u.Role,
		&u.StripeKey,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)

	return u, err
}
