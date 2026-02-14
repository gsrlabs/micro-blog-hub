package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/config"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/model"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/repository"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface{
	Register(ctx context.Context, req *model.CreateUserRequest) (uuid.UUID, error)
	Login(ctx context.Context, req *model.LoginRequest) (string, error) // Возвращает токен
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

type authService struct {
	repo repository.AuthRepository
	logger *zap.Logger
	cfg    *config.Config
}

func NewAuthService(repo repository.AuthRepository, logger *zap.Logger, cfg *config.Config) AuthService {
	return &authService{repo: repo, logger: logger, cfg: cfg}
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

func (s *authService) Login(ctx context.Context, req *model.LoginRequest) (string, error) {
	// 1. Ищем пользователя по email
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Специально возвращаем общую ошибку, чтобы не подсказывать хакерам (есть такой юзер или нет)
		s.logger.Warn("login failed: user not found", zap.String("email", req.Email))
		return "", fmt.Errorf("invalid credentials")
	}

	// 2. Проверяем пароль (сравниваем хеш из БД и присланный пароль)
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		s.logger.Warn("login failed: invalid password", zap.String("email", req.Email))
		return "", fmt.Errorf("invalid credentials")
	}

	// 3. Генерируем JWT токен
	expirationTime := time.Now().Add(time.Duration(s.cfg.JWT.ExpirationHours) * time.Hour)
	
	claims := &model.UserClaims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "auth-service",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	// Подписываем токен секретным ключом
	tokenString, err := token.SignedString([]byte(s.cfg.JWT.Secret))
	if err != nil {
		s.logger.Error("failed to generate token", zap.Error(err))
		return "", fmt.Errorf("failed to generate token")
	}

	s.logger.Info("user logged in", zap.String("user_id", user.ID.String()))
	return tokenString, nil
}

func (s *authService) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	user, err:= s.repo.GetByID(ctx, id)
	if err != nil{
		return nil, err
	}

	s.logger.Info("user found", zap.String("username", user.ID.String()), zap.String("id", id.String()))
	return user, nil
}

func (s *authService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err:= s.repo.GetByEmail(ctx, email)
	if err != nil{
		return nil, err
	}

	s.logger.Info("user found", zap.String("username", user.Username), zap.String("email", email))
	return user, nil
}