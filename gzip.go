package azure

import (
	"bytes"
	"compress/gzip"
	"strings"
	"sync"

	"github.com/Payel-git-ol/azure/ultrahttp"
)

// Пул gzip писателей для переиспользования
var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(nil)
	},
}

// GzipConfig конфигурация Gzip middleware
type GzipConfig struct {
	Level        int      // Уровень сжатия (1-9, -1 = default)
	MinSize      int      // Минимальный размер для сжатия (в байтах)
	ContentTypes []string // Типы контента для сжатия
}

// GzipMiddleware создаёт middleware для gzip сжатия
func GzipMiddleware(config GzipConfig) Middleware {
	if config.Level <= 0 || config.Level > 9 {
		config.Level = gzip.DefaultCompression
	}
	if config.MinSize <= 0 {
		config.MinSize = 256 // Не сжимать ответы меньше 256 байт
	}
	if len(config.ContentTypes) == 0 {
		config.ContentTypes = []string{
			"text/plain",
			"text/html",
			"text/css",
			"text/javascript",
			"application/json",
			"application/xml",
			"application/javascript",
		}
	}

	return func(c *Context, next ultrahttp.RouteHandler) {
		// Проверяем, поддерживает ли клиент gzip
		acceptEncoding := c.ultra.GetHeader("Accept-Encoding")
		if !strings.Contains(acceptEncoding, "gzip") {
			next(c.ultra)
			return
		}

		// Вызываем следующий хендлер
		next(c.ultra)

		// Получаем Content-Type ответа
		contentType := c.ultra.Response.Headers["Content-Type"]

		// Проверяем, нужно ли сжимать этот тип контента
		shouldCompress := false
		for _, ct := range config.ContentTypes {
			if strings.Contains(contentType, ct) {
				shouldCompress = true
				break
			}
		}

		if !shouldCompress {
			return
		}

		// Проверяем минимальный размер
		if len(c.ultra.Response.Body) < config.MinSize {
			return
		}

		// Сжимаем тело ответа
		var buf bytes.Buffer
		gz := gzipWriterPool.Get().(*gzip.Writer)
		gz.Reset(&buf)

		_, err := gz.Write(c.ultra.Response.Body)
		if err != nil {
			gz.Close()
			gzipWriterPool.Put(gz)
			return
		}

		gz.Close()
		gzipWriterPool.Put(gz)

		// Устанавливаем сжатое тело и заголовки
		c.ultra.Response.Body = buf.Bytes()
		c.ultra.Response.Headers["Content-Encoding"] = "gzip"
		c.ultra.Response.Headers["Vary"] = "Accept-Encoding"

		// Обновляем Content-Length
		delete(c.ultra.Response.Headers, "Content-Length")
	}
}

// Gzip создаёт простой Gzip middleware с настройками по умолчанию
func Gzip() Middleware {
	return GzipMiddleware(GzipConfig{})
}
