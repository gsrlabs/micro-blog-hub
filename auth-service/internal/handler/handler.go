package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/service"
	"go.uber.org/zap"
)



type AuthHandler struct {
	service service.AuthService
	logger *zap.Logger
}

func NewAuthHandler(s service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{service: s, logger: logger}
}

func (h *AuthHandler) Create(c *gin.Context) {
	//TODO
}

