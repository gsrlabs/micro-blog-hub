package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type AuthRepository interface {
	Create(ctx context.Context, user *model.User) (uuid.UUID, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateProfile(ctx context.Context, id uuid.UUID, username string) error
	UpdateEmail(ctx context.Context, id uuid.UUID, email string) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, newHash string) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetUsers(ctx context.Context, limit, offset int) ([]*model.User, error)
}

type authRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

var (
	ErrNotFound          = errors.New("user not found")
	ErrDuplicateUsername = errors.New("username already taken")
	ErrDuplicateEmail    = errors.New("email already taken")
)

func NewAuthRepository(pool *pgxpool.Pool, logger *zap.Logger) AuthRepository {
	return &authRepo{pool: pool, logger: logger}

}

func (r *authRepo) Create(ctx context.Context, user *model.User) (uuid.UUID, error) {
	query := `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	var id uuid.UUID
	err := r.pool.QueryRow(ctx, query, user.Username, user.Email, user.Password).Scan(&id)
	if err != nil {
		r.logger.Error("failed to insert user", zap.Error(err), zap.String("email", user.Email))
		return uuid.Nil, fmt.Errorf("insert user: %w", err)
	}

	return id, nil
}

func (r *authRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at 
		FROM users 
		WHERE id = $1
	`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *authRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at 
		FROM users 
		WHERE email = $1
	`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err // Тут можно проверить на pgx.ErrNoRows
	}
	return user, nil
}

func (r *authRepo) UpdateProfile(ctx context.Context, id uuid.UUID, username string) error {
	query := `UPDATE users SET username = $1, updated_at = NOW() WHERE id = $2`

	cmd, err := r.pool.Exec(ctx, query, username, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateUsername
		}
		return fmt.Errorf("db update profile: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *authRepo) UpdateEmail(ctx context.Context, id uuid.UUID, email string) error {
	query := `UPDATE users SET email = $1, updated_at = NOW() WHERE id = $2`

	cmd, err := r.pool.Exec(ctx, query, email, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("db update email: %w", err)
	}

	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *authRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, newHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`

	cmd, err := r.pool.Exec(ctx, query, newHash, userID)
	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *authRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *authRepo) GetUsers(ctx context.Context, limit, offset int) ([]*model.User, error) {
	query := `
		SELECT id, username, email, created_at, updated_at 
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*model.User, 0)
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, &u)
	}
	return result, nil
}
