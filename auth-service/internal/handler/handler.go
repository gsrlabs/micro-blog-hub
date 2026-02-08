package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/model"
	"github.com/gsrlabs/micro-blog-hub/auth-service/internal/service"
	"go.uber.org/zap"
)

type AuthHandler struct {
	service   service.AuthService
	logger    *zap.Logger
	validator *model.Validator
}

func NewAuthHandler(s service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{
		service:   s,
		logger:    logger,
		validator: model.NewValidator(), // Инициализируем
	}
}

var (
	ErrNotFound = errors.New("user not found")
)

func (h *AuthHandler) Create(c *gin.Context) {
	var req model.CreateUserRequest

	// 1. Логируем входящий запрос (лучше делать в middleware, но для примера тут)
	// h.logger.Debug("Attempting to create user")

	if err := c.ShouldBindJSON(&req); err != nil {
		// WARN: Ошибка валидации - это не ошибка сервера, это ошибка клиента
		h.logger.Warn("Failed to bind user JSON",
			zap.String("ip", c.ClientIP()),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.validator.ValidateStruct(&req); err != nil {
		// Можно сделать красивый вывод ошибок валидации
		h.logger.Warn("validation failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed", "details": err.Error()})
		return
	}

	id, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		// ERROR: Что-то сломалось внутри (БД, логика)
		h.logger.Error("Failed to create user service",
			zap.String("username", req.Username), // Логируем контекст!
			zap.String("email", req.Email),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// INFO: Успешная операция
	h.logger.Info("User created successfully",
		zap.String("user_id", id.String()),
	)

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *AuthHandler) GetByID(c *gin.Context) {
    idStr := c.Param("id")

    // 1. Валидация формата UUID (используем ту же библиотеку, что и в моделях)
    uid, err := uuid.Parse(idStr)
    if err != nil {
        h.logger.Warn("invalid uuid format", zap.String("id", idStr))
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id format"})
        return
    }

    // 2. Передаем уже типизированный uuid.UUID в сервис
    user, err := h.service.GetByID(c.Request.Context(), uid)
    if err != nil {
        // Проверяем, это ошибка "не найдено" или системный сбой
        if errors.Is(err, ErrNotFound) {
            c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
            return
        }
        
        h.logger.Error("failed to get user", zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
        return
    }

    c.JSON(http.StatusOK, model.ToResponse(user))
}

func (h *AuthHandler) GetByEmail(c *gin.Context) {
	email := c.Query("email") // Берем email из параметров строки ?email=...
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}

	user, err := h.service.GetByEmail(c.Request.Context(), email)
	if err != nil {
		h.logger.Warn("user not found", zap.String("email", email), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, model.ToResponse(user))
}
