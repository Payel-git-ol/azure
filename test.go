package azure

import (
	"bytes"
	"net/http/httptest"

	"github.com/Payel-git-ol/azure/ultrahttp"
)

// TestContext создаёт тестовый контекст для юнит-тестов
// Возвращает контекст и recorder для проверки ответа
func TestContext(method, path string, body []byte, headers map[string]string) (*Context, *httptest.ResponseRecorder) {
	// Создаём тестовый запрос
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Создаём recorder для записи ответа
	w := httptest.NewRecorder()

	// Создаём ultrahttp контекст
	ultraCtx := &ultrahttp.Context{
		Request: ultrahttp.Request{},
		Response: ultrahttp.Response{
			Status:     200,
			StatusText: "OK",
			Headers:    make(map[string]string),
			Body:       make([]byte, 0),
		},
	}

	// Копируем заголовки из запроса
	for k := range req.Header {
		ultraCtx.Request.Headers[k] = req.Header.Get(k)
	}

	// Парсим query string
	ultraCtx.Request.QueryString = []byte(req.URL.RawQuery)
	ultraCtx.Request.Method = []byte(method)
	ultraCtx.Request.Path = []byte(path)
	ultraCtx.Request.RemoteAddr = "127.0.0.1:12345"
	ultraCtx.SetParams(ultrahttp.NewRouteParams())

	// Создаём Azure контекст
	ctx := &Context{
		ultra: ultraCtx,
	}

	return ctx, w
}

// TestContextWithParams создаёт тестовый контекст с параметрами пути
func TestContextWithParams(method, path string, body []byte, headers map[string]string, params map[string]string) (*Context, *httptest.ResponseRecorder) {
	ctx, w := TestContext(method, path, body, headers)

	// Добавляем параметры пути
	if params != nil {
		rp := ctx.ultra.GetParams()
		for k, v := range params {
			rp.Keys = append(rp.Keys, k)
			rp.Values = append(rp.Values, v)
		}
	}

	return ctx, w
}

// GetResponseJSON получает JSON ответ из recorder
func GetResponseJSON(w *httptest.ResponseRecorder) []byte {
	return w.Body.Bytes()
}

// GetResponseString получает строковый ответ из recorder
func GetResponseString(w *httptest.ResponseRecorder) string {
	return w.Body.String()
}

// GetResponseStatus получает статус ответа
func GetResponseStatus(w *httptest.ResponseRecorder) int {
	return w.Code
}

// GetResponseHeader получает заголовок ответа
func GetResponseHeader(w *httptest.ResponseRecorder, key string) string {
	return w.Header().Get(key)
}

// AssertJSON сравнивает ожидаемый JSON с полученным
func AssertJSON(expected, actual []byte) bool {
	return bytes.Equal(expected, actual)
}

// AssertStatus проверяет статус ответа
func AssertStatus(w *httptest.ResponseRecorder, expected int) bool {
	return w.Code == expected
}

// AssertHeader проверяет заголовок ответа
func AssertHeader(w *httptest.ResponseRecorder, key, expected string) bool {
	return w.Header().Get(key) == expected
}
