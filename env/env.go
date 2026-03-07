// Package env - загрузка .env файлов без внешних зависимостей
// Простой и быстрый парсер с поддержкой комментариев и экспорта
package env

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

// variables хранит загруженные переменные окружения
var variables = make(map[string]string)

// Load загружает .env файл
// Если файл не найден - не ошибка (используются переменные из OS)
//
// Пример:
//
//	env.Load(".env")
//	env.Load() // загрузит .env из текущей директории
func Load(paths ...string) error {
	if len(paths) == 0 {
		paths = []string{".env"}
	}

	for _, path := range paths {
		if err := loadFile(path); err != nil {
			// Игнорируем отсутствие файла
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
	}

	return nil
}

// loadFile загружает один файл
func loadFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Пропускаем пустые строки и комментарии
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Поддерживаем export VAR=value
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimPrefix(line, "export ")
		}

		// Парсим VAR=value
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Удаляем кавычки если есть
		value = unquote(value)

		// Сохраняем переменную
		variables[key] = value

		// Устанавливаем в OS окружение
		os.Setenv(key, value)
	}

	return scanner.Err()
}

// unquote удаляет кавычки вокруг значения
func unquote(s string) string {
	if len(s) < 2 {
		return s
	}

	// Проверяем кавычки
	if (s[0] == '"' && s[len(s)-1] == '"') ||
		(s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}

	return s
}

// Get получает переменную окружения
// Возвращает значение и флаг существования
//
// Пример:
//
//	val, ok := env.Get("PORT")
func Get(key string) (string, bool) {
	// Сначала ищем в загруженных
	if val, ok := variables[key]; ok {
		return val, true
	}

	// Потом в OS окружении
	val, ok := os.LookupEnv(key)
	return val, ok
}

// MustGet получает переменную или возвращает значение по умолчанию
//
// Пример:
//
//	port := env.MustGet("PORT", "8080")
func MustGet(key, defaultValue string) string {
	if val, ok := Get(key); ok {
		return val
	}
	return defaultValue
}

// GetOr получает переменную или значение по умолчанию (алиас MustGet)
func GetOr(key, defaultValue string) string {
	return MustGet(key, defaultValue)
}

// Int получает переменную как int
// Возвращает значение и флаг успеха
//
// Пример:
//
//	port, ok := env.Int("PORT")
func Int(key string) (int, bool) {
	val, ok := Get(key)
	if !ok {
		return 0, false
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		return 0, false
	}

	return i, true
}

// MustInt получает переменную как int или значение по умолчанию
//
// Пример:
//
//	port := env.MustInt("PORT", 8080)
func MustInt(key string, defaultValue int) int {
	if val, ok := Int(key); ok {
		return val
	}
	return defaultValue
}

// IntOr алиас для MustInt
func IntOr(key string, defaultValue int) int {
	return MustInt(key, defaultValue)
}

// Int64 получает переменную как int64
func Int64(key string) (int64, bool) {
	val, ok := Get(key)
	if !ok {
		return 0, false
	}

	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false
	}

	return i, true
}

// MustInt64 получает переменную как int64 или значение по умолчанию
func MustInt64(key string, defaultValue int64) int64 {
	if val, ok := Int64(key); ok {
		return val
	}
	return defaultValue
}

// Float получает переменную как float64
func Float(key string) (float64, bool) {
	val, ok := Get(key)
	if !ok {
		return 0, false
	}

	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, false
	}

	return f, true
}

// MustFloat получает переменную как float64 или значение по умолчанию
func MustFloat(key string, defaultValue float64) float64 {
	if val, ok := Float(key); ok {
		return val
	}
	return defaultValue
}

// Bool получает переменную как bool
// Поддерживает: true/false, 1/0, yes/no, on/off
func Bool(key string) (bool, bool) {
	val, ok := Get(key)
	if !ok {
		return false, false
	}

	// Нормализуем
	val = strings.ToLower(strings.TrimSpace(val))

	switch val {
	case "true", "1", "yes", "on":
		return true, true
	case "false", "0", "no", "off":
		return false, true
	default:
		return false, false
	}
}

// MustBool получает переменную как bool или значение по умолчанию
func MustBool(key string, defaultValue bool) bool {
	if val, ok := Bool(key); ok {
		return val
	}
	return defaultValue
}

// All возвращает все загруженные переменные
func All() map[string]string {
	result := make(map[string]string, len(variables))
	for k, v := range variables {
		result[k] = v
	}
	return result
}

// Has проверяет наличие переменной
func Has(key string) bool {
	_, ok := Get(key)
	return ok
}
