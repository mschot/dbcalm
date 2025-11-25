package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type userRepository struct {
	db *DB
}

func NewUserRepository(db *DB) repository.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO user (username, password, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		user.Username,
		user.Password,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT username, password, created_at, updated_at
		FROM user
		WHERE username = ?
	`
	var user domain.User
	err := r.db.GetContext(ctx, &user, query, username)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", username)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE user
		SET password = ?, updated_at = ?
		WHERE username = ?
	`
	result, err := r.db.ExecContext(ctx, query,
		user.Password,
		user.UpdatedAt,
		user.Username,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found: %s", user.Username)
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, username string) error {
	query := `DELETE FROM user WHERE username = ?`
	result, err := r.db.ExecContext(ctx, query, username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found: %s", username)
	}

	return nil
}

func (r *userRepository) List(ctx context.Context) ([]*domain.User, error) {
	query := `
		SELECT username, password, created_at, updated_at
		FROM user
		ORDER BY username
	`
	var users []*domain.User
	err := r.db.SelectContext(ctx, &users, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}
