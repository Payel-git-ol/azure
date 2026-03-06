package ultrahttp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// HTTPClient - быстрый HTTP клиент для исходящих запросов
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient создаёт новый HTTP клиент
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Клиент по умолчанию
var defaultClient = NewHTTPClient(30 * time.Second)

// Get отправляет GET запрос
func (c *HTTPClient) Get(url string) ([]byte, error) {
	resp, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// Post отправляет POST запрос с JSON
func (c *HTTPClient) Post(url string, data interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	return io.ReadAll(resp.Body)
}

// PostRaw отправляет POST запрос с сырыми данными
func (c *HTTPClient) PostRaw(url string, contentType string, body io.Reader) ([]byte, error) {
	resp, err := c.client.Post(url, contentType, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	return io.ReadAll(resp.Body)
}

// Do отправляет кастомный запрос
func (c *HTTPClient) Do(method, url string, body []byte, headers map[string]string) ([]byte, error) {
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	return io.ReadAll(resp.Body)
}

// === Функции уровня пакета ===

// Get отправляет GET запрос (использует клиент по умолчанию)
func Get(url string) ([]byte, error) {
	return defaultClient.Get(url)
}

// Post отправляет POST запрос с JSON (использует клиент по умолчанию)
func Post(url string, data interface{}) ([]byte, error) {
	return defaultClient.Post(url, data)
}

// Do отправляет кастомный запрос (использует клиент по умолчанию)
func Do(method, url string, body []byte, headers map[string]string) ([]byte, error) {
	return defaultClient.Do(method, url, body, headers)
}
