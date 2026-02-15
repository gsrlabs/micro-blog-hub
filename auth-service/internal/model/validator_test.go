package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Тестовая структура для проверки валидатора
type testEmailStruct struct {
	Email string `validate:"strict_email"`
}

func TestValidator(t *testing.T) {
	v := NewValidator()

	t.Run("Strict Email Validation", func(t *testing.T) {
		tests := []struct {
			name    string
			email   string
			isValid bool
		}{
			// --- Позитивные кейсы (Valid) ---
			{"Valid simple", "test@example.com", true},
			{"Valid with dots", "first.last@domain.io", true},
			{"Valid subdomains", "user@mail.sub.example.com", true},
			{"Valid with plus", "user+label@gmail.com", true},
			{"Valid with hyphen", "my-email@domain-name.com", true},
			{"Valid numeric domain", "admin@123.com", true},

			// --- Негативные кейсы: Структура (Invalid) ---
			{"Invalid no at", "example.com", false},
			{"Invalid multiple at", "user@at@domain.com", false},
			{"Invalid no domain", "test@", false},
			{"Invalid no user", "@domain.com", false},
			{"Invalid short", "a@b", false},
			{"Invalid empty", "", false},

			// --- Негативные кейсы: Точки (Invalid) ---
			{"Invalid double dot in domain", "test@example..com", false},
			{"Invalid double dot in user", "test..user@example.com", false},
			{"Invalid start with dot", ".user@example.com", false},
			{"Invalid end with dot", "user.@example.com", false},
			{"Invalid domain start dot", "user@.example.com", false},

			// --- Негативные кейсы: Домены (Invalid) ---
			{"Invalid domain trailing dot", "user@example.com.", false},
			{"Invalid TLD too short", "user@example.c", false},
			{"Invalid numeric TLD", "user@example.123", false},
			{"Invalid spaces", "user name@example.com", false},
			{"Invalid special chars", "user#name@example.com", false},

			// --- Негативные кейсы: Длина ---
			{"Too long total", strings.Repeat("a", 245) + "@example.com", false},     // > 254
			{"Too long local part", strings.Repeat("a", 65) + "@example.com", false}, // Local part > 64
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				s := testEmailStruct{Email: tt.email}
				err := v.ValidateStruct(s)
				if tt.isValid {
					assert.NoError(t, err, "Email %s should be valid", tt.email)
				} else {
					assert.Error(t, err, "Email %s should be invalid", tt.email)
				}
			})
		}
	})
}
