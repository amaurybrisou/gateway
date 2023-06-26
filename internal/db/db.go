package db

import (
	"context"
	"fmt"
	"time"

	"github.com/amaurybrisou/gateway/internal/db/models"
	coremodels "github.com/amaurybrisou/gateway/pkg/core/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
)

type Database struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Database {
	return &Database{db: db}
}

func (d Database) CreateService(ctx context.Context, s models.Service) (models.Service, error) {
	query := `
	INSERT INTO service (id, name, domain, prefix, host, required_roles, costs, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	ON CONFLICT (name) DO UPDATE
	SET domain = excluded.domain,
		prefix = excluded.prefix,
		host = excluded.host,
		required_roles = excluded.required_roles,
		costs = excluded.costs		
	RETURNING *;
	`

	err := d.db.QueryRow(ctx, query, s.ID, s.Name, s.Domain, s.Prefix, s.Host, s.RequiredRoles, s.Costs, s.CreatedAt).Scan(
		&s.ID, &s.Name, &s.Domain, &s.Prefix, &s.Host, &s.RequiredRoles, &s.Costs, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt)
	if err != nil {
		return models.Service{}, fmt.Errorf("failed to create service: %v", err)
	}

	return s, nil
}

func (d Database) DeleteService(ctx context.Context, serviceID uuid.UUID) (bool, error) {
	query := `
		DELETE FROM service
		WHERE id = $1`

	result, err := d.db.Exec(ctx, query, serviceID)
	if err != nil {
		return false, fmt.Errorf("failed to delete service: %v", err)
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected == 1, nil
}

func (d *Database) GetServiceByID(ctx context.Context, serviceID uuid.UUID) (models.Service, error) {
	query := `
		SELECT id, name, prefix, required_roles, costs, created_at, updated_at, deleted_at
		FROM "service"
		WHERE id = $1 AND deleted_at IS NULL
	`

	var service models.Service
	err := d.db.QueryRow(ctx, query, serviceID).Scan(
		&service.ID,
		&service.Name,
		&service.Prefix,
		&service.RequiredRoles,
		&service.Costs,
		&service.CreatedAt,
		&service.UpdatedAt,
		&service.DeletedAt,
	)
	if err != nil {
		// Handle the error (e.g., return an error response, log the error)
		return models.Service{}, err
	}

	return service, nil
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

func (d Database) AddRole(ctx context.Context, userID uuid.UUID, role models.Role, duration time.Duration) (models.UserRole, error) {
	s := models.UserRole{}
	query := `INSERT INTO user_role (user_id, role, expiration_time, created_at) VALUES ($1, $2, $3, $4)" RETURNING *`

	err := d.db.QueryRow(ctx, query, userID, role, duration).Scan(
		&s.UserID, &s.Role, &s.ExpirationTime, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt)
	if err != nil {
		return models.UserRole{}, fmt.Errorf("failed to create service: %v", err)
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

func (d Database) CreatePayment(ctx context.Context, userID, serviceID uuid.UUID, duration time.Duration, amount float32) (models.UserPayment, error) {
	query := `
		INSERT INTO user_payment (id, user_id, service_id, amount, duration, created_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING *`

	payment := models.UserPayment{
		ID:        uuid.New(),
		UserID:    userID,
		ServiceID: serviceID,
		Amount:    amount,
		Duration:  duration,
		CreatedAt: time.Now(),
	}

	err := d.db.QueryRow(ctx, query, payment.ID, payment.UserID, payment.ServiceID, payment.Amount, payment.Duration, payment.CreatedAt, payment.Status).Scan(
		&payment.ID, &payment.UserID, &payment.ServiceID, &payment.CreatedAt, &payment.Status, &payment.UpdatedAt, &payment.DeletedAt)
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
		SELECT * FROM user_payment`

	rows, err := d.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user payments: %v", err)
	}
	defer rows.Close()

	var payments []models.UserPayment
	for rows.Next() {
		var payment models.UserPayment
		if err := rows.Scan(
			&payment.ID, &payment.ServiceID, &payment.CreatedAt, &payment.Status, &payment.UpdatedAt, &payment.DeletedAt,
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
			&service.ID, &service.Name, &service.Prefix, &service.RequiredRoles, &service.Costs,
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

const (
	UniqueViolationErr = pq.ErrorCode("23505")
)

func IsErrorCode(err error, errcode pq.ErrorCode) bool {
	if pgerr, ok := err.(*pq.Error); ok {
		return pgerr.Code == errcode
	}
	return false
}

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
		tx.Rollback(ctx)
		return fmt.Errorf("failed to create user: %v", err)
	}

	// Create access token
	tokenQuery := `
		INSERT INTO access_token (user_id, external_id, token, expires_at)
		VALUES ($1, $2, $3, $4) ON CONFLICT(external_id) 
		DO UPDATE SET token = excluded.token, expires_at = excluded.expires_at`

	_, err = tx.Exec(ctx, tokenQuery, token.UserID, token.ExternalID, token.Token, token.ExpiresAt)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to create access token: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

func (d Database) GetServices(ctx context.Context) ([]models.Service, error) {
	query := `SELECT * FROM service`

	rows, err := d.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query services: %v", err)
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var service models.Service
		if err := rows.Scan(
			&service.ID,
			&service.Name,
			&service.Domain,
			&service.Prefix,
			&service.Host,
			&service.RequiredRoles,
			&service.Costs,
			&service.CreatedAt,
			&service.UpdatedAt,
			&service.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan service row: %v", err)
		}
		services = append(services, service)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred while reading services: %v", err)
	}

	return services, nil
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

func (d *Database) GetPaymentByID(ctx context.Context, paymentID uuid.UUID) (models.UserPayment, error) {
	query := `
		SELECT id, user_id, service_id, amount, duration, created_at, status, updated_at, deleted_at
		FROM "user_payment"
		WHERE id = $1
	`

	var payment models.UserPayment
	err := d.db.QueryRow(ctx, query, paymentID).Scan(
		&payment.ID,
		&payment.UserID,
		&payment.ServiceID,
		&payment.Amount,
		&payment.Duration,
		&payment.CreatedAt,
		&payment.Status,
		&payment.UpdatedAt,
		&payment.DeletedAt,
	)
	if err != nil {
		// Handle the error (e.g., return an error response, log the error)
		return models.UserPayment{}, err
	}

	return payment, nil
}
