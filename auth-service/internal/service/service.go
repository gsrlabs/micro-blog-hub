package service

import (
	"context"
	"fmt"

	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/model"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/repository"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
)

type AuthService interface{
	Register(ctx context.Context, req *model.CreateUserRequest) (uuid.UUID, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

type authService struct {
	repo repository.AuthRepository
	logger *zap.Logger
}

func NewAuthService(repo repository.AuthRepository, logger *zap.Logger) AuthService {
	return &authService{repo: repo, logger: logger}
}

func (s *authService) Register(ctx context.Context, req *model.CreateUserRequest) (uuid.UUID, error) {
	// 1. Хешируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, fmt.Errorf("hash password: %w", err)
	}

	// 2. Маппим в доменную модель
	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	// 3. Сохраняем в БД
	id, err := s.repo.Create(ctx, user)
	if err != nil {
		return uuid.Nil, err
	}

	s.logger.Info("user registered", zap.String("id", id.String()), zap.String("email", user.Email))
	return id, nil
}

func (s *authService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err:= s.repo.GetByEmail(ctx, email)
	if err != nil{
		return nil, err
	}

	s.logger.Info("user found", zap.String("username", user.Username), zap.String("email", email))
	return user, nil
}