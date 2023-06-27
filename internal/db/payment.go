package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/amaurybrisou/gateway/internal/db/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (d Database) CreatePayment(ctx context.Context, id, userID, planID uuid.UUID) (models.UserPayment, error) {
	query := `
		INSERT INTO user_payment (id, user_id, plan_id, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, plan_id, status, created_at, updated_at, deleted_at`

	payment := models.UserPayment{
		ID:     id,
		UserID: userID,
		PlanID: planID,
		Status: models.PaymentPending,
	}

	err := d.db.QueryRow(ctx, query, payment.ID, payment.UserID, payment.PlanID, payment.Status).Scan(
		&payment.ID,
		&payment.UserID,
		&payment.PlanID,
		&payment.Status,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&payment.DeletedAt,
	)
	if err != nil {
		return models.UserPayment{}, fmt.Errorf("failed to create payment: %v", err)
	}

	return payment, nil
}

func (d Database) UpdatePayment(ctx context.Context, paymentID uuid.UUID, status models.PaymentStatus) error {
	query := `
		UPDATE user_payment
		SET status = $1
		WHERE id = $2`

	result, err := d.db.Exec(ctx, query, status, paymentID)
	if err != nil {
		return fmt.Errorf("failed to update payment: %v", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected != 1 {
		return fmt.Errorf("no payment found with the given ID")
	}

	return nil
}

func (d Database) GetPayments(ctx context.Context) ([]models.UserPayment, error) {
	query := `
		SELECT id, user_id, plan_id, status,, created_at,  updated_at, deleted_at FROM user_payment`

	rows, err := d.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user payments: %v", err)
	}
	defer rows.Close()

	var payments []models.UserPayment
	for rows.Next() {
		var payment models.UserPayment
		if err := rows.Scan(
			&payment.ID,
			&payment.UserID,
			&payment.PlanID,
			&payment.Status,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&payment.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user payment: %v", err)
		}
		payments = append(payments, payment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over user payments: %v", err)
	}

	return payments, nil
}

func (d *Database) GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (models.UserPayment, error) {
	query := `
		SELECT id, user_id, plan_id, status, created_at, updated_at, deleted_at
		FROM "user_payment"
		WHERE id = $1
	`

	var payment models.UserPayment
	err := d.db.QueryRow(ctx, query, paymentID).Scan(
		&payment.ID,
		&payment.UserID,
		&payment.PlanID,
		&payment.Status,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&payment.DeletedAt,
	)
	if err != nil {
		// Handle the error (e.g., return an error response, log the error)
		return models.UserPayment{}, err
	}

	return payment, nil
}

func (d *Database) GetFullPaymentByID(ctx context.Context, paymentID uuid.UUID) (models.UserPayment, error) {
	query := `
		SELECT 
			up.id, up.user_id, up.plan_id, up.status, up.created_at, up.updated_at, up.deleted_at,
			p.id as p_id, p.name, p.description, p.price, p.currency, p.duration, p.service_id
		FROM user_payment up
		INNER JOIN plan p ON p.id = up.plan_id
		WHERE up.id = $1
	`

	payment := models.UserPayment{Plan: models.Plan{}}
	err := d.db.QueryRow(ctx, query, paymentID).Scan(
		&payment.ID,
		&payment.UserID,
		&payment.PlanID,
		&payment.Status,
		&payment.CreatedAt,
		&payment.UpdatedAt,
		&payment.DeletedAt,

		&payment.Plan.ID,
		&payment.Plan.Name,
		&payment.Plan.Description,
		&payment.Plan.Price,
		&payment.Plan.Currency,
		&payment.Plan.Duration,
		&payment.Plan.ServiceID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.UserPayment{}, nil
		}
		return models.UserPayment{}, err
	}

	return payment, nil
}
