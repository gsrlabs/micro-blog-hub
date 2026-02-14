package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)



type AuthRepository interface {
	Create(ctx context.Context, user *model.User) (uuid.UUID, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}


type authRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

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
