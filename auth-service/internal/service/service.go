package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/config"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/model"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/repository"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Register(ctx context.Context, req *model.CreateUserRequest) (uuid.UUID, error)
	Login(ctx context.Context, req *model.LoginRequest) (string, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	ChangeProfile(ctx context.Context, userID uuid.UUID, req *model.ChangeProfileRequest) error
	ChangeEmail(ctx context.Context, userID uuid.UUID, req *model.ChangeEmailRequest) error
	ChangePassword(ctx context.Context, userID uuid.UUID, req *model.ChangePasswordRequest) error
	Delete(ctx context.Context, userID uuid.UUID) error
	GetUsers(ctx context.Context, limit, offset int) ([]*model.User, error)
}

type authService struct {
	repo   repository.AuthRepository
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
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	s.logger.Info("user found", zap.String("username", user.ID.String()), zap.String("id", id.String()))
	return user, nil
}

func (s *authService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	s.logger.Info("user found", zap.String("username", user.Username), zap.String("email", email))
	return user, nil
}

func (s *authService) ChangeProfile(ctx context.Context, userID uuid.UUID, req *model.ChangeProfileRequest) error {
	// Вызываем правильный метод репозитория
	err := s.repo.UpdateProfile(ctx, userID, req.NewUsername)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateUsername) {
			return err
		}
		if errors.Is(err, repository.ErrNotFound) {
			return err
		}
		s.logger.Error("failed to update profile in db", zap.Error(err))
		return fmt.Errorf("internal error")
	}

	s.logger.Info("profile changed successfully", zap.String("user_id", userID.String()), zap.String("new_username", req.NewUsername))
	return nil
}

func (s *authService) ChangeEmail(ctx context.Context, userID uuid.UUID, req *model.ChangeEmailRequest) error {
	// Вызываем правильный метод репозитория
	err := s.repo.UpdateEmail(ctx, userID, req.NewEmail)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateEmail) {
			return err
		}
		if errors.Is(err, repository.ErrNotFound) {
			return err
		}
		s.logger.Error("failed to update email in db", zap.Error(err))
		return fmt.Errorf("internal error")
	}

	s.logger.Info("email changed successfully", zap.String("user_id", userID.String()), zap.String("new_email", req.NewEmail))
	return nil
}

func (s *authService) ChangePassword(ctx context.Context, userID uuid.UUID, req *model.ChangePasswordRequest) error {
	// 1. Получаем текущего пользователя из базы
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// 2. Проверяем, правильно ли введен СТАРЫЙ пароль
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword))
	if err != nil {
		s.logger.Warn("change password failed: wrong old password", zap.String("user_id", userID.String()))
		return fmt.Errorf("invalid old password")
	}

	// 3. Хешируем НОВЫЙ пароль
	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash new password", zap.Error(err))
		return fmt.Errorf("internal error")
	}

	// 4. Сохраняем новый хеш в базу
	err = s.repo.UpdatePassword(ctx, userID, string(newHash))
	if err != nil {
		s.logger.Error("failed to update password in db", zap.Error(err))
		return fmt.Errorf("internal error")
	}

	s.logger.Info("password changed successfully", zap.String("user_id", userID.String()))
	return nil
}

func (s *authService) Delete(ctx context.Context, userID uuid.UUID) error {
	err := s.repo.Delete(ctx, userID)
	if err != nil {
		return err
	}

	s.logger.Info("user has been deleted successfully", zap.String("userID", userID.String()))
	return nil
}

func (s *authService) GetUsers(ctx context.Context, limit, offset int) ([]*model.User, error) {
	users, err := s.repo.GetUsers(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	s.logger.Info("users found", zap.String("count=", strconv.Itoa(len(users))))
	return users, nil
}
