package db

import (
	"context"
	"fmt"
	"time"

	"github.com/amaurybrisou/gateway/internal/db/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

func (d Database) HasRole(ctx context.Context, id uuid.UUID, roles ...models.Role) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM user_role
			WHERE user_id = $1 AND role = ANY($2)
		)`

	var hasRole bool
	err := d.db.QueryRow(ctx, query, id, pq.Array(roles)).Scan(&hasRole)
	if err != nil {
		return false, fmt.Errorf("failed to check user role: %v", err)
	}

	return hasRole, nil
}

func (d Database) AddRole(ctx context.Context, userID uuid.UUID, role models.Role, expiresAt time.Time) (models.UserRole, error) {
	s := models.UserRole{}
	query := `INSERT INTO user_role (user_id, role, expiration_time) VALUES ($1, $2, $3) RETURNING *`

	err := d.db.QueryRow(ctx, query, userID, role, expiresAt).Scan(
		&s.UserID, &s.Role, &s.ExpirationTime, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt)
	if err != nil {
		return models.UserRole{}, fmt.Errorf("failed to add role: %v", err)
	}

	return s, nil
}

func (d Database) DelRole(ctx context.Context, userID uuid.UUID, role models.Role) (bool, error) {
	result, err := d.db.Exec(ctx, "DELETE FROM user_role WHERE user_id = $1 AND role = $2", userID, role)
	if err != nil {
		return false, fmt.Errorf("failed to prepare statement: %v", err)
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected == 1, nil
}

func (d Database) GetRoles(ctx context.Context) ([]models.Role, error) {
	query := `
		SELECT DISTINCT UNNEST(required_roles) FROM service`

	rows, err := d.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve roles: %v", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		if err := rows.Scan(&role); err != nil {
			return nil, fmt.Errorf("failed to scan role: %v", err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over roles: %v", err)
	}

	return roles, nil
}
