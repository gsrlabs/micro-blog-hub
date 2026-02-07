package service

import (
	//"github.com/gsrlabs/micro-blog-hub/auth-service/internal/model"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/repository"
	"go.uber.org/zap"
)

type AuthService interface{
	//TODO Create(ctx context.Context, auth *model.User) error

}

type authService struct {
	repo repository.AuthRepository
	logger *zap.Logger
}

func NewAuthService(repo repository.AuthRepository, logger *zap.Logger) AuthService {
	return &authService{repo: repo, logger: logger}
}