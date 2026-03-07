package ultrahttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// M - сокращение для map[string]interface{}
type M map[string]interface{}

// jsonBuffer - пул для буферов JSON
var jsonBuffer = &bytes.Buffer{}

// ParseJSON парсит JSON из байтов в структуру
func ParseJSON(data []byte, v interface{}) error {
	if len(data) == 0 {
		return json.Unmarshal([]byte("{}"), v)
	}
	return json.Unmarshal(data, v)
}

// FastMarshalJSON - быстрая сериализация JSON (экспортированная версия)
//
//go:noinline
func FastMarshalJSON(v interface{}) []byte {
	return fastMarshalJSON(v)
}

// FastMarshalM - быстрая сериализация map[string]interface{} (экспортированная версия)
//
//go:noinline
func FastMarshalM(m M) []byte {
	return fastMarshalM(m)
}

// MarshalJSON - алиас для FastMarshalJSON
func MarshalJSON(v interface{}) ([]byte, error) {
	return fastMarshalJSON(v), nil
}

// === БЫСТРЫЕ МЕТОДЫ ДЛЯ JSON (ОПТИМИЗИРОВАННЫЕ) ===

// fastMarshalJSON - быстрая сериализация для простых типов
//
//go:noinline
func fastMarshalJSON(v interface{}) []byte {
	// Специальный path для M типа (самый частый случай)
	if m, ok := v.(M); ok {
		return fastMarshalM(m)
	}

	// Fallback на стандартный json.Marshal
	buf, _ := json.Marshal(v)
	return buf
}

// fastMarshalM - быстрая сериализация map[string]interface{}
//
//go:noinline
func fastMarshalM(m M) []byte {
	if len(m) == 0 {
		return []byte("{}")
	}

	// Используем буфер для сборки JSON
	var buf [4096]byte
	pos := 0

	buf[pos] = '{'
	pos++

	first := true
	for k, v := range m {
		if !first {
			buf[pos] = ','
			pos++
		}
		first = false

		// Кодируем ключ (строка с квотами)
		buf[pos] = '"'
		pos++

		// Экранируем строку ключа (просто копируем если ASCII)
		for i := 0; i < len(k); i++ {
			if k[i] == '"' {
				buf[pos] = '\\'
				pos++
			} else if k[i] == '\\' {
				buf[pos] = '\\'
				pos++
			}
			buf[pos] = k[i]
			pos++
		}

		buf[pos] = '"'
		pos++
		buf[pos] = ':'
		pos++

		// Кодируем значение
		switch val := v.(type) {
		case string:
			buf[pos] = '"'
			pos++
			for i := 0; i < len(val); i++ {
				if val[i] == '"' {
					buf[pos] = '\\'
					pos++
				} else if val[i] == '\\' {
					buf[pos] = '\\'
					pos++
				}
				buf[pos] = val[i]
				pos++
			}
			buf[pos] = '"'
			pos++

		case int:
			pos += appendIntFast(buf[pos:], val)

		case int64:
			pos += appendInt64(buf[pos:], val)

		case float64:
			// Быстрое преобразование float64 в строку
			if val == float64(int64(val)) {
				pos += appendInt64(buf[pos:], int64(val))
			} else {
				s := fmt.Sprintf("%g", val)
				pos += copy(buf[pos:], s)
			}

		case bool:
			if val {
				pos += copy(buf[pos:], []byte("true"))
			} else {
				pos += copy(buf[pos:], []byte("false"))
			}

		case nil:
			pos += copy(buf[pos:], []byte("null"))

		default:
			// Для других типов используем json.Marshal
			b, _ := json.Marshal(v)
			pos += copy(buf[pos:], b)
		}
	}

	buf[pos] = '}'
	pos++

	return buf[:pos]
}

// appendInt64 - добавляет int64 в буфер без allocations
//
//go:noinline
func appendInt64(buf []byte, n int64) int {
	if n == 0 {
		buf[0] = '0'
		return 1
	}

	if n < 0 {
		buf[0] = '-'
		n = -n
		pos := 1

		digits := [20]byte{}
		i := 0
		for n > 0 {
			digits[i] = '0' + byte(n%10)
			n /= 10
			i++
		}
		for j := i - 1; j >= 0; j-- {
			buf[pos] = digits[j]
			pos++
		}
		return pos
	}

	digits := [20]byte{}
	i := 0
	for n > 0 {
		digits[i] = '0' + byte(n%10)
		n /= 10
		i++
	}
	pos := 0
	for j := i - 1; j >= 0; j-- {
		buf[pos] = digits[j]
		pos++
	}
	return pos
}

// appendIntFast - добавляет int в буфер без allocations
//
//go:noinline
func appendIntFast(buf []byte, n int) int {
	return appendInt64(buf, int64(n))
}

//go:noinline
func (c *Context) SetJSON(data interface{}) {
	c.Response.Headers["Content-Type"] = "application/json; charset=utf-8"

	// Используем быстрый path для пустого объекта
	if data == nil {
		c.Response.Body = append(c.Response.Body[:0], "null"...)
		return
	}

	// Используем оптимизированный маршал
	buf := fastMarshalJSON(data)
	c.Response.Body = append(c.Response.Body[:0], buf...)
}

//go:noinline
func (c *Context) SetJSONStatus(status int, data interface{}) {
	c.Response.Status = status
	c.Response.Headers["Content-Type"] = "application/json; charset=utf-8"

	if data == nil {
		c.Response.Body = append(c.Response.Body[:0], "null"...)
		return
	}

	buf := fastMarshalJSON(data)
	c.Response.Body = append(c.Response.Body[:0], buf...)
}

//go:noinline
func (c *Context) SetJSONBytes(data []byte) {
	c.Response.Headers["Content-Type"] = "application/json; charset=utf-8"
	c.Response.Body = append(c.Response.Body[:0], data...)
}

//go:noinline
func (c *Context) SetHTML(html string) {
	c.Response.Headers["Content-Type"] = "text/html; charset=utf-8"
	c.Response.Body = append(c.Response.Body[:0], html...)
}

//go:noinline
func (c *Context) SetHTMLStatus(status int, html string) {
	c.Response.Status = status
	c.Response.Headers["Content-Type"] = "text/html; charset=utf-8"
	c.Response.Body = append(c.Response.Body[:0], html...)
}

//go:noinline
func (c *Context) SetText(text string) {
	c.Response.Headers["Content-Type"] = "text/plain; charset=utf-8"
	c.Response.Body = append(c.Response.Body[:0], text...)
}

// === КВАЗИ-ROUTERS ===

// RouteMatcher функция для матчинга путей
type RouteMatcher func(path []byte) bool

// ExactMatch точное совпадение пути
func ExactMatch(match []byte) RouteMatcher {
	return func(path []byte) bool {
		return bytesEqual(path, match)
	}
}

// PrefixMatch совпадение по префиксу
func PrefixMatch(prefix []byte) RouteMatcher {
	return func(path []byte) bool {
		if len(path) < len(prefix) {
			return false
		}
		return bytesEqual(path[:len(prefix)], prefix)
	}
}

// === ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ДЛЯ ЗАГОЛОВКОВ ===

// GetHeader получает заголовок из запроса
func (c *Context) GetHeader(key string) string {
	return c.Request.Headers[key]
}

// SetHeader устанавливает заголовок ответа
func (c *Context) SetHeader(key, value string) {
	c.Response.Headers[key] = value
}

// SetStatus устанавливает статус ответа
func (c *Context) SetStatus(status int, text string) {
	c.Response.Status = status
	c.Response.StatusText = text
}

// SetBody устанавливает тело ответа
func (c *Context) SetBody(body []byte) {
	c.Response.Body = append(c.Response.Body[:0], body...)
}

// SetBodyString устанавливает тело ответа из строки
func (c *Context) SetBodyString(body string) {
	c.Response.Body = append(c.Response.Body[:0], body...)
}

// GetContentType получает Content-Type из запроса
func (c *Context) GetContentType() string {
	return c.GetHeader("Content-Type")
}

// GetContentLength получает Content-Length из запроса
func (c *Context) GetContentLength() int {
	return c.Request.ContentLen
}

// GetAuthorization получает Authorization заголовок
func (c *Context) GetAuthorization() string {
	return c.GetHeader("Authorization")
}

// GetUserAgent получает User-Agent заголовок
func (c *Context) GetUserAgent() string {
	return c.GetHeader("User-Agent")
}

// GetReferer получает Referer заголовок
func (c *Context) GetReferer() string {
	return c.GetHeader("Referer")
}

// GetHost получает Host заголовок
func (c *Context) GetHost() string {
	return c.GetHeader("Host")
}

// === COOKIE ===

// SetCookie устанавливает cookie
func (c *Context) SetCookie(name, value string) {
	c.Response.Headers["Set-Cookie"] = name + "=" + value + "; Path=/; HttpOnly"
}

// SetCookieFull устанавливает cookie с параметрами
func (c *Context) SetCookieFull(name, value, path, domain string, maxAge int, secure, httpOnly bool) {
	cookie := name + "=" + value
	if path != "" {
		cookie += "; Path=" + path
	} else {
		cookie += "; Path=/"
	}
	if domain != "" {
		cookie += "; Domain=" + domain
	}
	if maxAge > 0 {
		cookie += "; Max-Age=" + itoa(maxAge)
	}
	if secure {
		cookie += "; Secure"
	}
	if httpOnly {
		cookie += "; HttpOnly"
	}
	c.Response.Headers["Set-Cookie"] = cookie
}

// GetCookie получает cookie по имени из запроса
func (c *Context) GetCookie(name string) (string, bool) {
	cookieHeader := c.GetHeader("Cookie")
	if cookieHeader == "" {
		return "", false
	}

	// Парсим cookie: name1=value1; name2=value2
	pairs := strings.Split(cookieHeader, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		idx := strings.Index(pair, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(pair[:idx])
		value := strings.TrimSpace(pair[idx+1:])
		if key == name {
			return value, true
		}
	}
	return "", false
}

// BindJSON парсит JSON из тела запроса в структуру
//
//go:noinl
//go:noinline
func (c *Context) BindJSON(v interface{}) error {
	body := c.Request.Body
	if len(body) == 0 {
		return nil
	}
	return ParseJSON(body, v)
}

// === QUERY PARAMS ===

// GetQueryParam получает параметр из query string
func (c *Context) GetQueryParam(key string) string {
	query := b2s(c.Request.QueryString)
	if query == "" {
		return ""
	}

	pairs := strings.Split(query, "&")
	for _, pair := range pairs {
		idx := strings.Index(pair, "=")
		if idx < 0 {
			continue
		}
		k := pair[:idx]
		v := pair[idx+1:]
		if k == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// === REDIRECT ===

// Redirect делает редирект
func (c *Context) Redirect(url string, status int) {
	if status == 0 {
		status = 302
	}
	c.Response.Status = status
	c.Response.StatusText = statusText(status)
	c.Response.Headers["Location"] = url
	c.Response.Body = c.Response.Body[:0]
}

func statusText(status int) string {
	switch status {
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found"
	case 303:
		return "See Other"
	case 307:
		return "Temporary Redirect"
	case 308:
		return "Permanent Redirect"
	default:
		return "Redirect"
	}
}

// === CORS ===

// EnableCORS включает CORS для всех origin
func (c *Context) EnableCORS() {
	c.Response.Headers["Access-Control-Allow-Origin"] = "*"
	c.Response.Headers["Access-Control-Allow-Methods"] = "GET, POST, PUT, DELETE, PATCH, OPTIONS"
	c.Response.Headers["Access-Control-Allow-Headers"] = "Content-Type, Authorization, X-Requested-With"
	c.Response.Headers["Access-Control-Max-Age"] = "86400"
}

// EnableCORSWithOrigin включает CORS для конкретного origin
func (c *Context) EnableCORSWithOrigin(origin string) {
	c.Response.Headers["Access-Control-Allow-Origin"] = origin
	c.Response.Headers["Access-Control-Allow-Methods"] = "GET, POST, PUT, DELETE, PATCH, OPTIONS"
	c.Response.Headers["Access-Control-Allow-Headers"] = "Content-Type, Authorization, X-Requested-With"
}

// === Методы проверки ===

// IsGET проверяет что метод GET
func (c *Context) IsGET() bool {
	return bytesEqual(c.Request.Method, methodGET)
}

// IsPOST проверяет что метод POST
func (c *Context) IsPOST() bool {
	return bytesEqual(c.Request.Method, methodPOST)
}

// IsPUT проверяет что метод PUT
func (c *Context) IsPUT() bool {
	return bytesEqual(c.Request.Method, methodPUT)
}

// IsDELETE проверяет что метод DELETE
func (c *Context) IsDELETE() bool {
	return bytesEqual(c.Request.Method, methodDELETE)
}

// IsOPTIONS проверяет что метод OPTIONS
func (c *Context) IsOPTIONS() bool {
	return bytesEqual(c.Request.Method, methodOPTIONS)
}

// IsPATCH проверяет что метод PATCH
func (c *Context) IsPATCH() bool {
	return bytesEqual(c.Request.Method, methodPATCH)
}

// IsHEAD проверяет что метод HEAD
func (c *Context) IsHEAD() bool {
	return bytesEqual(c.Request.Method, methodHEAD)
}

// === БЫСТРЫЕ ОТВЕТЫ ===

// SendOK отправляет 200 OK
func (c *Context) SendOK() {
	c.Response.Status = 200
	c.Response.StatusText = "OK"
	c.Response.Body = c.Response.Body[:0]
}

// SendNotFound отправляет 404
func (c *Context) SendNotFound() {
	c.Response.Status = 404
	c.Response.StatusText = "Not Found"
	c.Response.Body = append(c.Response.Body[:0], "404 Not Found"...)
}

// SendBadRequest отправляет 400
func (c *Context) SendBadRequest() {
	c.Response.Status = 400
	c.Response.StatusText = "Bad Request"
	c.Response.Body = append(c.Response.Body[:0], "400 Bad Request"...)
}

// SendServerError отправляет 500
func (c *Context) SendServerError() {
	c.Response.Status = 500
	c.Response.StatusText = "Internal Server Error"
	c.Response.Body = append(c.Response.Body[:0], "500 Internal Server Error"...)
}

// SendNoContent отправляет 204 No Content
func (c *Context) SendNoContent() {
	c.Response.Status = 204
	c.Response.StatusText = "No Content"
	c.Response.Body = c.Response.Body[:0]
}

// === ВСПОМОГАТЕЛЬНЫЕ ===

// itoa конвертирует int в string
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var buf [32]byte
	i := len(buf)

	negative := n < 0
	if negative {
		n = -n
	}

	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}

	if negative {
		i--
		buf[i] = '-'
	}

	return string(buf[i:])
}

// PathEquals сравнивает путь с строкой
func (c *Context) PathEquals(s string) bool {
	return bytesEqual(c.Request.Path, []byte(s))
}

// MethodEquals сравнивает метод с строкой
func (c *Context) MethodEquals(s string) bool {
	return bytesEqual(c.Request.Method, []byte(s))
}
