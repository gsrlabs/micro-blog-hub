package model

import (
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Выносим регулярное выражение в переменную уровня пакета.
// MustCompile вызовет панику при старте, если регулярка кривая (это хорошо для отлова ошибок).
var emailRegex = regexp.MustCompile(`^(?P<local>[a-zA-Z0-9._%+\-]+)@(?P<domain>([a-zA-Z0-9\-]+\.)+[a-zA-Z]{2,})$`)

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

func validateEmail(fl validator.FieldLevel) bool {
    email := fl.Field().String()
    
    // 1. Общая длина
    if len(email) < 3 || len(email) > 254 {
        return false
    }

    // 2. Проверка регуляркой
    if !emailRegex.MatchString(email) {
        return false
    }

    // 3. Дополнительно: длина локальной части (до @)
    parts := strings.Split(email, "@")
    if len(parts[0]) > 64 {
        return false
    }

    // 4. Проверка на двойные точки в локальной части (регулярка выше это не всегда ловит)
    if strings.Contains(parts[0], "..") || strings.HasPrefix(parts[0], ".") || strings.HasSuffix(parts[0], ".") {
        return false
    }

    return true
}