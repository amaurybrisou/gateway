package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/amaurybrisou/gateway/src/database/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
)

const (
	serviceSelectFields     = "id, name, description, prefix, domain, host, image_url, status, required_roles"
	serviceSelectFieldsFull = "id, name, description, prefix, domain, host, image_url, status, required_roles, pricing_table_key, pricing_table_publishable_key, created_at, updated_at, deleted_at, required_roles = '{}' as has_access"
	serviceInsertFields     = "id, name, description, prefix, domain, host, image_url, status, required_roles, pricing_table_key, pricing_table_publishable_key, created_at"
)

func (d Database) CreateService(ctx context.Context, s models.Service) (models.Service, error) {
	query := `
	INSERT INTO service (` + serviceInsertFields + `)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	ON CONFLICT (name) DO UPDATE
	SET domain = excluded.domain,
		prefix = excluded.prefix,
		host = excluded.host,
		required_roles = excluded.required_roles,
		image_url = excluded.image_url,
		description = excluded.description,
		pricing_table_key = excluded.pricing_table_key,
		pricing_table_publishable_key = excluded.pricing_table_publishable_key
	RETURNING ` + serviceSelectFieldsFull

	row := d.db.QueryRow(
		ctx,
		query,
		s.ID,
		s.Name,
		s.Description,
		s.Prefix,
		s.Domain,
		s.Host,
		s.ImageURL,
		"ADDED",
		pq.Array(s.RequiredRoles),
		s.PricingTableKey,
		s.PricingTablePublishableKey,
		time.Now(),
	)

	s, err := scanServiceFull(row)
	if err != nil {
		return models.Service{}, fmt.Errorf("failed to create service: %w", err)
	}

	return s, nil
}

func (d Database) DeleteService(ctx context.Context, serviceID uuid.UUID) (bool, error) {
	query := `
		DELETE FROM service
		WHERE id = $1`

	result, err := d.db.Exec(ctx, query, serviceID)
	if err != nil {
		return false, fmt.Errorf("failed to delete service: %w", err)
	}

	rowsAffected := result.RowsAffected()
	return rowsAffected == 1, nil
}

func (d *Database) GetServiceByID(ctx context.Context, serviceID uuid.UUID) (models.Service, error) {
	query := `
		SELECT ` + serviceSelectFieldsFull + `
		FROM "service"
		WHERE id = $1 AND deleted_at IS NULL
	`

	row := d.db.QueryRow(ctx, query, serviceID)

	service, err := scanServiceFull(row)
	if err != nil {
		// Handle the error (e.g., return an error response, log the error)
		return models.Service{}, err
	}

	return service, nil
}

func (d *Database) GetServiceByName(ctx context.Context, serviceName string) (models.Service, error) {
	query := `
		SELECT ` + serviceSelectFieldsFull + `
		FROM service
		WHERE name = $1 AND deleted_at IS NULL
	`

	row := d.db.QueryRow(ctx, query, serviceName)

	service, err := scanServiceFull(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Service{}, nil
		}
		return models.Service{}, err
	}

	return service, nil
}

func (d Database) GetServices(ctx context.Context) ([]*models.Service, error) {
	query := `SELECT ` + serviceSelectFieldsFull + ` FROM service WHERE deleted_at IS NULL`

	rows, err := d.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query services: %w", err)
	}
	defer rows.Close()

	var services []*models.Service
	for rows.Next() {
		service, err := scanServiceFull(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service row: %w", err)
		}
		services = append(services, &service)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred while reading services: %w", err)
	}

	return services, nil
}

func (d *Database) GetServiceByPrefixOrDomain(ctx context.Context, prefix, domain string) (models.Service, error) {
	query := `
        SELECT ` + serviceSelectFields + `
        FROM service
        WHERE prefix = $1 OR domain = $2
        LIMIT 1
    `

	row := d.db.QueryRow(ctx, query, prefix, domain)

	service, err := scanService(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return service, fmt.Errorf("service not found")
		}
		return service, err
	}

	return service, nil
}

func (d *Database) UpdateServiceStatus(ctx context.Context, serviceID uuid.UUID, status string) error {
	query := `
        UPDATE service
        SET status = $1
        WHERE id = $2
    `

	_, err := d.db.Exec(ctx, query, status, serviceID)
	if err != nil {
		return fmt.Errorf("failed to update service status: %w", err)
	}

	return nil
}

func scanService(row localRow) (models.Service, error) {
	var service models.Service
	var nullImage sql.NullString

	err := row.Scan(
		&service.ID,
		&service.Name,
		&service.Description,
		&service.Prefix,
		&service.Domain,
		&service.Host,
		&nullImage,
		&service.Status,
		&service.RequiredRoles,
	)
	if err != nil {
		return models.Service{}, fmt.Errorf("failed to scan service row: %w", err)
	}

	if nullImage.Valid {
		service.ImageURL = &nullImage.String
	}

	return service, nil
}

func scanServiceFull(row localRow) (models.Service, error) {
	var service models.Service
	var nullImage sql.NullString
	err := row.Scan(
		&service.ID,
		&service.Name,
		&service.Description,
		&service.Prefix,
		&service.Domain,
		&service.Host,
		&nullImage,
		&service.Status,
		&service.RequiredRoles,
		&service.PricingTableKey,
		&service.PricingTablePublishableKey,
		&service.CreatedAt,
		&service.UpdatedAt,
		&service.DeletedAt,
		&service.HasAccess,
	)
	if err != nil {
		return models.Service{}, fmt.Errorf("failed to scan service row: %w", err)
	}

	if nullImage.Valid {
		service.ImageURL = &nullImage.String
	}
	return service, nil
}
