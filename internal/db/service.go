package db

import (
	"context"
	"fmt"

	"github.com/amaurybrisou/gateway/internal/db/models"
	"github.com/google/uuid"
)

func (d Database) CreateService(ctx context.Context, s models.Service) (models.Service, error) {
	query := `
	INSERT INTO service (id, name, domain, prefix, host, required_roles, pricing_table_key,
		pricing_table_publishable_key, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	ON CONFLICT (name) DO UPDATE
	SET domain = excluded.domain,
		prefix = excluded.prefix,
		host = excluded.host,
		required_roles = excluded.required_roles,
		pricing_table_key = excluded.pricing_table_key,
		pricing_table_publishable_key = excluded.pricing_table_publishable_key
	RETURNING *;
	`

	err := d.db.QueryRow(ctx, query, s.ID, s.Name, s.Domain, s.Prefix, s.Host, s.RequiredRoles, s.PricingTableKey, s.PricingTablePublishableKey, s.CreatedAt).Scan(
		&s.ID, &s.Name, &s.Domain, &s.Prefix, &s.Host, &s.RequiredRoles, &s.PricingTableKey, &s.PricingTablePublishableKey, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt)
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
		SELECT id, name, prefix, required_roles, pricing_table_key, pricing_table_publishable_key, created_at, updated_at, deleted_at
		FROM "service"
		WHERE id = $1 AND deleted_at IS NULL
	`

	var service models.Service
	err := d.db.QueryRow(ctx, query, serviceID).Scan(
		&service.ID,
		&service.Name,
		&service.Prefix,
		&service.RequiredRoles,
		&service.PricingTableKey,
		&service.PricingTablePublishableKey,
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

func (d *Database) GetServiceByName(ctx context.Context, serviceName string) (models.Service, error) {
	query := `
		SELECT id, name, prefix, required_roles, pricing_table_key, pricing_table_publishable_key, created_at, updated_at, deleted_at
		FROM service
		WHERE name = $1 AND deleted_at IS NULL
	`

	var service models.Service
	err := d.db.QueryRow(ctx, query, serviceName).Scan(
		&service.ID,
		&service.Name,
		&service.Prefix,
		&service.RequiredRoles,
		&service.PricingTableKey,
		&service.PricingTablePublishableKey,
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
			&service.PricingTableKey,
			&service.PricingTablePublishableKey,
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
