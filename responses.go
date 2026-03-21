package azure

import (
	"encoding/xml"
	"fmt"

	"github.com/goccy/go-yaml"
)

// XML отправляет XML ответ
func (c *Context) XML(data interface{}) {
	c.ultra.SetHeader("Content-Type", "application/xml; charset=utf-8")

	buf, err := xml.Marshal(data)
	if err != nil {
		c.JsonStatus(500, M{
			"error": "XML marshal error: " + err.Error(),
		})
		return
	}

	c.ultra.SetBody(buf)
}

// XMLStatus отправляет XML ответ со статусом
func (c *Context) XMLStatus(status int, data interface{}) {
	c.ultra.SetStatus(status, "")
	c.ultra.SetHeader("Content-Type", "application/xml; charset=utf-8")

	buf, err := xml.Marshal(data)
	if err != nil {
		c.JsonStatus(500, M{
			"error": "XML marshal error: " + err.Error(),
		})
		return
	}

	c.ultra.SetBody(buf)
}

// XMLBytes отправляет XML ответ из байтов
func (c *Context) XMLBytes(data []byte) {
	c.ultra.SetHeader("Content-Type", "application/xml; charset=utf-8")
	c.ultra.SetBody(data)
}

// YAML отправляет YAML ответ
func (c *Context) YAML(data interface{}) {
	c.ultra.SetHeader("Content-Type", "application/yaml; charset=utf-8")

	buf, err := yaml.Marshal(data)
	if err != nil {
		c.JsonStatus(500, M{
			"error": "YAML marshal error: " + err.Error(),
		})
		return
	}

	c.ultra.SetBody(buf)
}

// YAMLStatus отправляет YAML ответ со статусом
func (c *Context) YAMLStatus(status int, data interface{}) {
	c.ultra.SetStatus(status, "")
	c.ultra.SetHeader("Content-Type", "application/yaml; charset=utf-8")

	buf, err := yaml.Marshal(data)
	if err != nil {
		c.JsonStatus(500, M{
			"error": "YAML marshal error: " + err.Error(),
		})
		return
	}

	c.ultra.SetBody(buf)
}

// YAMLBytes отправляет YAML ответ из байтов
func (c *Context) YAMLBytes(data []byte) {
	c.ultra.SetHeader("Content-Type", "application/yaml; charset=utf-8")
	c.ultra.SetBody(data)
}

// Text отправляет текстовый ответ
func (c *Context) Text(format string, args ...interface{}) {
	c.ultra.SetHeader("Content-Type", "text/plain; charset=utf-8")

	var text string
	if len(args) > 0 {
		text = sprintf(format, args...)
	} else {
		text = format
	}

	c.ultra.SetBody([]byte(text))
}

// TextStatus отправляет текстовый ответ со статусом
func (c *Context) TextStatus(status int, format string, args ...interface{}) {
	c.ultra.SetStatus(status, "")
	c.ultra.SetHeader("Content-Type", "text/plain; charset=utf-8")

	var text string
	if len(args) > 0 {
		text = sprintf(format, args...)
	} else {
		text = format
	}

	c.ultra.SetBody([]byte(text))
}

// String отправляет строковый ответ (алиас для Text)
func (c *Context) String(format string, args ...interface{}) {
	c.Text(format, args...)
}

// StringStatus отправляет строковый ответ со статусом (алиас для TextStatus)
func (c *Context) StringStatus(status int, format string, args ...interface{}) {
	c.TextStatus(status, format, args...)
}

// sprintf простая реализация для форматирования строк
func sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// BindXML парсит XML из тела запроса в структуру
func (c *Context) BindXML(v interface{}) error {
	body := c.GetBody()
	if len(body) == 0 {
		return nil
	}
	return xml.Unmarshal(body, v)
}

// BindYAML парсит YAML из тела запроса в структуру
func (c *Context) BindYAML(v interface{}) error {
	body := c.GetBody()
	if len(body) == 0 {
		return nil
	}
	return yaml.Unmarshal(body, v)
}
