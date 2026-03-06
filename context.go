package azure

import "github.com/Payel-git-ol/azure/ultrahttp"

// Context контекст запроса
type Context struct {
	ultra *ultrahttp.Context
}

// Json отправляет JSON ответ
func (c *Context) Json(data M) {
	c.ultra.SetJSON(data)
}

// JsonStatus отправляет JSON ответ со статусом
func (c *Context) JsonStatus(status int, data M) {
	c.ultra.SetJSONStatus(status, data)
}

// Html отправляет HTML ответ
func (c *Context) Html(html string) {
	c.ultra.SetHTML(html)
}

// HtmlStatus отправляет HTML ответ со статусом
func (c *Context) HtmlStatus(status int, html string) {
	c.ultra.SetStatus(status, "")
	c.ultra.SetHTML(html)
}

// Send отправляет данные
func (c *Context) Send(data []byte) {
	c.ultra.SetBody(data)
}

// SetStatus устанавливает статус
func (c *Context) SetStatus(code int, text string) {
	c.ultra.SetStatus(code, text)
}

// Param получает параметр пути
func (c *Context) Param(key string) string {
	// TODO: реализовать когда будет поддержка параметров в роутере
	return ""
}

// GetBody получает тело запроса
func (c *Context) GetBody() []byte {
	return c.ultra.GetBody()
}

// GetHeader получает заголовок
func (c *Context) GetHeader(key string) string {
	return c.ultra.GetHeader(key)
}

// SetHeader устанавливает заголовок ответа
func (c *Context) SetHeader(key, value string) {
	c.ultra.SetHeader(key, value)
}

// SetCookie устанавливает cookie
func (c *Context) SetCookie(name, value string) {
	c.ultra.SetCookie(name, value)
}

// GetCookie получает cookie
func (c *Context) GetCookie(name string) (string, bool) {
	return c.ultra.GetCookie(name)
}

// GetQueryParam получает query параметр
func (c *Context) GetQueryParam(key string) string {
	return c.ultra.GetQueryParam(key)
}

// BindJSON парсит JSON из тела запроса в структуру
func (c *Context) BindJSON(v interface{}) error {
	return c.ultra.BindJSON(v)
}
