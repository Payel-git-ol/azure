package azure

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

// Get отправляет GET запрос и возвращает строку
func (c *HTTPClient) GetString(url string) (string, error) {
	resp, err := c.Get(url)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

// Post отправляет POST запрос с JSON и возвращает строку
func (c *HTTPClient) PostString(url string, data interface{}) (string, error) {
	resp, err := c.Post(url, data)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

// Do отправляет кастомный запрос и возвращает строку
func (c *HTTPClient) DoString(method, url string, body []byte, headers map[string]string) (string, error) {
	resp, err := c.Do(method, url, body, headers)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

// === Функции уровня пакета ===

// Get отправляет GET запрос
func Get(url string) ([]byte, error) {
	return defaultClient.Get(url)
}

// GetString отправляет GET запрос и возвращает строку
func GetString(url string) (string, error) {
	return defaultClient.GetString(url)
}

// Post отправляет POST запрос с JSON
func Post(url string, data interface{}) ([]byte, error) {
	return defaultClient.Post(url, data)
}

// PostString отправляет POST запрос с JSON и возвращает строку
func PostString(url string, data interface{}) (string, error) {
	return defaultClient.PostString(url, data)
}

// Do отправляет кастомный запрос
func Do(method, url string, body []byte, headers map[string]string) ([]byte, error) {
	return defaultClient.Do(method, url, body, headers)
}

// DoString отправляет кастомный запрос и возвращает строку
func DoString(method, url string, body []byte, headers map[string]string) (string, error) {
	return defaultClient.DoString(method, url, body, headers)
}
