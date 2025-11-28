package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type authCodeRepository struct {
	db *DB
}

func NewAuthCodeRepository(db *DB) repository.AuthCodeRepository {
	return &authCodeRepository{db: db}
}

func (r *authCodeRepository) Create(ctx context.Context, authCode *domain.AuthCode) error {
	scopesJSON, err := json.Marshal(authCode.Scopes)
	if err != nil {
		return fmt.Errorf("failed to marshal scopes: %w", err)
	}

	query := `
		INSERT INTO auth_code (code, username, scopes, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err = r.db.ExecContext(ctx, query,
		authCode.Code,
		authCode.Username,
		string(scopesJSON),
		authCode.ExpiresAt,
		authCode.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create auth code: %w", err)
	}
	return nil
}

func (r *authCodeRepository) FindByCode(ctx context.Context, code string) (*domain.AuthCode, error) {
	query := `
		SELECT code, username, scopes, expires_at, created_at
		FROM auth_code
		WHERE code = ?
	`
	var authCode domain.AuthCode
	var scopesJSON string
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&authCode.Code,
		&authCode.Username,
		&scopesJSON,
		&authCode.ExpiresAt,
		&authCode.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("auth code not found: %s", code)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find auth code: %w", err)
	}

	if err := json.Unmarshal([]byte(scopesJSON), &authCode.Scopes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal scopes: %w", err)
	}

	return &authCode, nil
}

func (r *authCodeRepository) Delete(ctx context.Context, code string) error {
	query := `DELETE FROM auth_code WHERE code = ?`
	result, err := r.db.ExecContext(ctx, query, code)
	if err != nil {
		return fmt.Errorf("failed to delete auth code: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("auth code not found: %s", code)
	}

	return nil
}

func (r *authCodeRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM auth_code WHERE expires_at < ?`
	_, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired auth codes: %w", err)
	}
	return nil
}
