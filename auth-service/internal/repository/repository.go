package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)



type AuthRepository interface {
	//TODO
}

var ErrNotFound = errors.New("user not found")

type authRepo struct {
	pool *pgxpool.Pool
	logger *zap.Logger
}

func NewAuthRepository(pool *pgxpool.Pool, logger *zap.Logger) AuthRepository{
	return &authRepo{pool: pool, logger: logger}

}

