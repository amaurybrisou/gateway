package db

import (
	"context"
	"fmt"

	"github.com/amaurybrisou/gateway/internal/db/models"
	"github.com/google/uuid"
)

func (d Database) CreateAccessToken(ctx context.Context, t models.AccessToken) (models.AccessToken, error) {
	query := `
		INSERT INTO access_token (user_id, token, expires_at)
		VALUES ($1, $2, $3)
		RETURNING *`

	err := d.db.QueryRow(ctx, query, t.UserID, t.Token, t.ExpiresAt).Scan(
		&t.UserID, &t.Token, &t.ExpiresAt)
	if err != nil {
		return models.AccessToken{}, fmt.Errorf("failed to create access token: %v", err)
	}

	return t, nil
}

func (d *Database) DeleteAccessToken(ctx context.Context, userID uuid.UUID) error {
	query := `
		DELETE FROM access_token WHERE user_id = $1
	`

	_, err := d.db.Exec(ctx, query, userID)
	if err != nil {
		return err
	}

	return nil
}

func (d Database) HasToken(ctx context.Context, token string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM access_token
			WHERE token = $1
			AND expires_at > now()
		)`

	var hasToken bool
	err := d.db.QueryRow(ctx, query, token).Scan(&hasToken)
	if err != nil {
		return false, fmt.Errorf("failed to check access token: %v", err)
	}

	return hasToken, nil
}
