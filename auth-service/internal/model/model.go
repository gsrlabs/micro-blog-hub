package model

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID
	Username  string
	Email     string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateUserRequest struct {
    Username string `json:"username" validate:"required,min=2,max=50"`
    Email    string `json:"email" validate:"required,strict_email"` 
    Password string `json:"password" validate:"required,min=8,max=72"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}


// LoginRequest - то, что шлет клиент
type LoginRequest struct {
	Email    string `json:"email" validate:"required,strict_email"`
	Password string `json:"password" validate:"required"`
}

// UserClaims - расширяем стандартный токен своими полями
type UserClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	jwt.RegisteredClaims
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}

type ChangeProfileRequest struct {
	NewUsername string `json:"new_username" validate:"required,min=2,max=50"`
}

type ChangeEmailRequest struct {
	NewEmail string `json:"new_email" validate:"required,strict_email"` 
}
