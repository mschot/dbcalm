package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type clientRepository struct {
	db *DB
}

func NewClientRepository(db *DB) repository.ClientRepository {
	return &clientRepository{db: db}
}

func (r *clientRepository) Create(ctx context.Context, client *domain.Client) error {
	scopesJSON, err := json.Marshal(client.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	query := `
		INSERT INTO client (id, secret, label, scopes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.ExecContext(ctx, query,
		client.ID,
		client.Secret,
		client.Label,
		string(scopesJSON),
		client.CreatedAt,
		client.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	return nil
}

func (r *clientRepository) FindByID(ctx context.Context, id string) (*domain.Client, error) {
	query := `
		SELECT id, secret, label, scopes, created_at, updated_at
		FROM client
		WHERE id = ?
	`
	var client domain.Client
	var scopesJSON string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&client.ID,
		&client.Secret,
		&client.Label,
		&scopesJSON,
		&client.CreatedAt,
		&client.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("client not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find client: %w", err)
	}

	if err := json.Unmarshal([]byte(scopesJSON), &client.Scopes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
	}

	return &client, nil
}

func (r *clientRepository) Update(ctx context.Context, client *domain.Client) error {
	scopesJSON, err := json.Marshal(client.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	query := `
		UPDATE client
		SET label = ?, scopes = ?, updated_at = ?
		WHERE id = ?
	`
	result, err := r.db.ExecContext(ctx, query,
		client.Label,
		string(scopesJSON),
		client.UpdatedAt,
		client.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update client: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("client not found: %s", client.ID)
	}

	return nil
}

func (r *clientRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM client WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("client not found: %s", id)
	}

	return nil
}

func (r *clientRepository) List(ctx context.Context) ([]*domain.Client, error) {
	query := `
		SELECT id, secret, label, scopes, created_at, updated_at
		FROM client
		ORDER BY label
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}
	defer rows.Close()

	var clients []*domain.Client
	for rows.Next() {
		var client domain.Client
		var scopesJSON string
		err := rows.Scan(
			&client.ID,
			&client.Secret,
			&client.Label,
			&scopesJSON,
			&client.CreatedAt,
			&client.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client: %w", err)
		}

		if err := json.Unmarshal([]byte(scopesJSON), &client.Scopes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
		}

		clients = append(clients, &client)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating clients: %w", err)
	}

	return clients, nil
}
