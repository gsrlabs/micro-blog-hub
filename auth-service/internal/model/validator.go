package model

import (
	"regexp"
	"strings"
	
	"github.com/go-playground/validator/v10"
)

// Validate is a global instance of the validator used to check struct tags across the application.
var Validate *validator.Validate

// init initializes the global validator and registers custom validation rules.
func init() {
	Validate = validator.New()
	//_ = Validate.RegisterValidation("email", validateEmail)
}

func validateEmail(fl validator.FieldLevel) bool {
	value := fl.Field().String()

	// 1. Базовая проверка длины
    if len(value) < 3 || len(value) > 254 {
        return false
    }
    
    // 2. Проверка на наличие символа @
    atIndex := strings.Index(value, "@")
    if atIndex == -1 || atIndex == 0 || atIndex == len(value)-1 {
        return false
    }
    
    // 3. Разделяем на локальную часть и домен
    localPart := value[:atIndex]
    domainPart := value[atIndex+1:]
    
    // 4. Проверка локальной части (до @)
    if len(localPart) > 64 {
        return false
    }
    
    // 5. Проверка доменной части
    if len(domainPart) > 253 {
        return false
    }
    
    // 6. Регулярное выражение для основной валидации
    pattern := `^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`
    
    re := regexp.MustCompile(pattern)
    if !re.MatchString(value) {
        return false
    }
    
    // 7. Проверка, что домен имеет хотя бы одну точку
    if !strings.Contains(domainPart, ".") {
        return false
	}
    // 8. Проверка, что последняя часть домена не слишком короткая
    lastDotIndex := strings.LastIndex(domainPart, ".")
    if lastDotIndex == -1 || len(domainPart[lastDotIndex+1:]) < 2 {
        return false
    }
    
    return true

}