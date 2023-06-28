package database

import (
	"context"
	"fmt"
	"time"

	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (d Database) HasRole(ctx context.Context, id uuid.UUID, roles ...models.Role) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM user_role
			WHERE user_id = $1 
			AND (
				(role = ANY($2) AND (expires_at IS NULL OR expires_at > now()) AND deleted_at IS NULL) OR
				($2 = '{}')
			)
		)`

	var hasRole bool
	err := d.db.QueryRow(ctx, query, id, pq.Array(roles)).Scan(&hasRole)
	if err != nil {
		return false, fmt.Errorf("failed to check user role: %w", err)
	}

	return hasRole, nil
}

func (d Database) AddRole(ctx context.Context, userID uuid.UUID, subID string, role models.Role, expiresAt *time.Time) (models.UserRole, error) {
	s := models.UserRole{}
	query := `INSERT INTO user_role (user_id, subscription_id, role, expires_at) VALUES ($1, $2, $3, $4) 
	ON CONFLICT(user_id, role) DO UPDATE SET subscription_id = excluded.subscription_id, expires_at = excluded.expires_at, deleted_at = NULL
	RETURNING user_id, subscription_id, role, expires_at, created_at, updated_at, deleted_at`

	err := d.db.QueryRow(ctx, query, userID, subID, role, expiresAt).Scan(
		&s.UserID, &s.SubscriptionID, &s.Role, &s.ExpiresAt, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt)
	if err != nil {
		return models.UserRole{}, fmt.Errorf("failed to add role: %w", err)
	}

	return s, nil
}

func (d Database) DelRole(ctx context.Context, userID uuid.UUID, role models.Role) (bool, error) {
	result, err := d.db.Exec(ctx, "UPDATE user_role SET deleted_at = now() WHERE user_id = $1 AND role = $2", userID, role)
	if err != nil {
		return false, fmt.Errorf("failed to prepare statement: %w", err)
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected == 1, nil
}

func (d Database) DelRoleBySubscriptionID(ctx context.Context, subID string) (bool, error) {
	result, err := d.db.Exec(ctx, "UPDATE user_role SET deleted_at = now() WHERE subscription_id = $1", subID)
	if err != nil {
		return false, fmt.Errorf("failed to prepare statement: %w", err)
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected == 1, nil
}

func (d Database) UpdateRoleExpiration(ctx context.Context, subID string, expiresAt *time.Time) (bool, error) {
	result, err := d.db.Exec(ctx, "UPDATE user_role SET deleted_at = null, expires_at = $2 WHERE subscription_id = $1", subID, expiresAt)
	if err != nil {
		return false, fmt.Errorf("failed to prepare statement: %w", err)
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected == 1, nil
}

func (d Database) GetRoles(ctx context.Context) ([]models.Role, error) {
	query := `
		SELECT DISTINCT UNNEST(required_roles) FROM service`

	rows, err := d.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve roles: %w", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over roles: %w", err)
	}

	return roles, nil
}
