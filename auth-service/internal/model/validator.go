package model

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// Выносим регулярное выражение в переменную уровня пакета.
// MustCompile вызовет панику при старте, если регулярка кривая (это хорошо для отлова ошибок).
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Validator - обертка над библиотекой валидации
type Validator struct {
	validate *validator.Validate
}

// NewValidator создает новый экземпляр
func NewValidator() *Validator {
	v := validator.New()
	
	// Регистрируем наш кастомный валидатор
	// Назовем его "strict_email", чтобы отличать от встроенного
	_ = v.RegisterValidation("strict_email", validateEmail)
	
	return &Validator{validate: v}
}

// ValidateStruct - метод для проверки структур
func (v *Validator) ValidateStruct(s interface{}) error {
	return v.validate.Struct(s)
}

// validateEmail - оптимизированная функция
func validateEmail(fl validator.FieldLevel) bool {
	email := fl.Field().String()
    
	// 1. Быстрая проверка длины (RFC 5321)
	if len(email) < 3 || len(email) > 254 {
		return false
	}

	// 2. Проверка регулярным выражением (уже скомпилированным!)
	return emailRegex.MatchString(email)
}