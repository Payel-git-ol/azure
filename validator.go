package azure

import (
	"github.com/go-playground/validator/v10"
)

// validate - глобальный экземпляр валидатора
var validate *validator.Validate

func init() {
	validate = validator.New()
}

// Validator - валидатор для структур
type Validator struct {
	validator *validator.Validate
}

// NewValidator создаёт новый валидатор
func NewValidator() *Validator {
	return &Validator{
		validator: validator.New(),
	}
}

// Validate проверяет структуру на валидность
func (v *Validator) Validate(i interface{}) error {
	return v.validator.Struct(i)
}

// ValidateWithContext проверяет структуру с контекстом
func (v *Validator) ValidateWithContext(i interface{}, ctx interface{}) error {
	// Используем стандартный Struct без контекста для простоты
	return v.validator.Struct(i)
}

// RegisterValidation регистрирует кастомную валидацию
func (v *Validator) RegisterValidation(tag string, fn validator.Func) error {
	return v.validator.RegisterValidation(tag, fn)
}

// RegisterTagNameFunc регистрирует функцию для получения имени поля
func (v *Validator) RegisterTagNameFunc(fn validator.TagNameFunc) {
	v.validator.RegisterTagNameFunc(fn)
}

// GlobalValidate выполняет глобальную валидацию
func GlobalValidate(i interface{}) error {
	return validate.Struct(i)
}

// BindAndValidate парсит JSON и валидирует структуру
func (c *Context) BindAndValidate(v interface{}) error {
	if err := c.BindJSON(v); err != nil {
		return err
	}
	return validate.Struct(v)
}

// ValidationErrors ошибки валидации
type ValidationErrors struct {
	Errors map[string]string `json:"errors"`
}

// GetValidationErrors получает ошибки валидации в удобном формате
func GetValidationErrors(err error) *ValidationErrors {
	if err == nil {
		return nil
	}

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return &ValidationErrors{
			Errors: map[string]string{"error": err.Error()},
		}
	}

	errors := make(map[string]string)
	for _, e := range validationErrs {
		key := e.Field()
		switch e.Tag() {
		case "required":
			errors[key] = "Это поле обязательно для заполнения"
		case "email":
			errors[key] = "Некорректный email адрес"
		case "url":
			errors[key] = "Некорректный URL"
		case "min":
			errors[key] = "Значение должно быть минимум " + e.Param()
		case "max":
			errors[key] = "Значение должно быть максимум " + e.Param()
		case "min_len":
			errors[key] = "Минимальная длина " + e.Param()
		case "max_len":
			errors[key] = "Максимальная длина " + e.Param()
		case "len":
			errors[key] = "Должно быть длиной " + e.Param()
		case "oneof":
			errors[key] = "Должно быть одним из: " + e.Param()
		case "numeric":
			errors[key] = "Должно быть числом"
		case "alpha":
			errors[key] = "Должно содержать только буквы"
		case "alphanum":
			errors[key] = "Должно содержать только буквы и цифры"
		default:
			errors[key] = "Некорректное значение"
		}
	}

	return &ValidationErrors{
		Errors: errors,
	}
}

// SendValidationErrors отправляет ошибки валидации в ответе
func (c *Context) SendValidationErrors(err error) {
	validationErrs := GetValidationErrors(err)
	c.JsonStatus(400, M{
		"error":  "Validation failed",
		"errors": validationErrs.Errors,
	})
}
