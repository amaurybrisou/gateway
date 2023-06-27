package db

import (
	"context"

	"github.com/amaurybrisou/gateway/internal/db/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// CreatePlan creates a new plan in the database.
func (d *Database) CreatePlan(ctx context.Context, serviceID uuid.UUID, name string, description string, price float64, duration models.SubDuration, currency string) (models.Plan, error) {
	plan := models.Plan{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		Price:       price,
		Currency:    currency,
		Duration:    duration,
		ServiceID:   serviceID,
	}

	query := `
		INSERT INTO "plan" ("id", "name", "description", "price", "currency", "duration", "service_id")
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING *
	`

	err := d.db.QueryRow(ctx, query, plan.ID, plan.Name, plan.Description, plan.Price, plan.Currency, plan.Duration, plan.ServiceID).
		Scan(
			&plan.ID,
			&plan.Name,
			&plan.Description,
			&plan.Price,
			&plan.Currency,
			&plan.Duration,
			&plan.ServiceID,
		)
	if err != nil {
		// Handle the error according to your application's requirements
		return models.Plan{}, err
	}

	return plan, nil
}

func (d Database) GetPlanByID(ctx context.Context, planID uuid.UUID) (models.Plan, error) {
	query := "SELECT id, name, description, price, currency, duration, service_id FROM plan WHERE id = $1"
	row := d.db.QueryRow(ctx, query, planID)

	var plan models.Plan
	err := row.Scan(&plan.ID, &plan.Name, &plan.Description, &plan.Price, &plan.Currency, &plan.Duration, &plan.ServiceID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.Plan{}, nil // Plan not found
		}
		return models.Plan{}, err
	}

	return plan, nil
}
