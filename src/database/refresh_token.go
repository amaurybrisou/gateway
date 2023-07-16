package database

import (
	"context"
)

// GetRefreshTokensByUserID retrieves all refresh tokens associated with a user ID.
func (r Database) GetRefreshTokenByUserID(ctx context.Context, userID string) (token string, err error) {
	row := r.db.QueryRow(ctx, "SELECT refresh_token FROM refresh_tokens WHERE user_id = $1", userID)
	if err = row.Scan(&token); err != nil {
		return "", err
	}

	return token, nil
}

// AddRefreshToken adds a new refresh token for a user ID.
func (r Database) AddRefreshToken(ctx context.Context, userID, refreshToken string) error {
	_, err := r.db.Exec(ctx, "INSERT INTO refresh_tokens (user_id, refresh_token) VALUES ($1, $2)", userID, refreshToken)
	return err
}

// RemoveRefre removes a refresh token for a user ID.
func (r Database) RemoveRefreshToken(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM refresh_tokens WHERE user_id = $1", userID)
	return err
}
