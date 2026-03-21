package azure

import (
	"html/template"
	"io"
	"path/filepath"
	"sync"

	"github.com/Payel-git-ol/azure/ultrahttp"
)

// TemplateConfig конфигурация шаблонов
type TemplateConfig struct {
	Glob       string           // Glob паттерн для файлов шаблонов
	LeftDelim  string           // Левый разделитель
	RightDelim string           // Правый разделитель
	FuncMap    template.FuncMap // Кастомные функции
}

// Template рендерер шаблонов
type Template struct {
	templates *template.Template
	config    TemplateConfig
	mu        sync.RWMutex
}

// NewTemplate создаёт новый рендерер шаблонов
func NewTemplate(config TemplateConfig) (*Template, error) {
	t := &Template{
		config: config,
	}

	if config.LeftDelim != "" {
		config.LeftDelim = "{{"
	}
	if config.RightDelim != "" {
		config.RightDelim = "}}"
	}

	// Загружаем шаблоны
	if err := t.Load(); err != nil {
		return nil, err
	}

	return t, nil
}

// Load загружает шаблоны из файлов
func (t *Template) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	files, err := filepath.Glob(t.config.Glob)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	tmpl := template.New("")
	if t.config.FuncMap != nil {
		tmpl.Funcs(t.config.FuncMap)
	}
	tmpl.Delims(t.config.LeftDelim, t.config.RightDelim)

	var errLoad error
	t.templates, errLoad = tmpl.ParseFiles(files...)
	return errLoad
}

// Reload перезагружает шаблоны
func (t *Template) Reload() error {
	return t.Load()
}

// Render рендерит шаблон в writer
func (t *Template) Render(w io.Writer, name string, data interface{}) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.templates == nil {
		_, err := w.Write([]byte("Templates not loaded"))
		return err
	}

	return t.templates.ExecuteTemplate(w, name, data)
}

// RenderToString рендерит шаблон в строку
func (t *Template) RenderToString(name string, data interface{}) (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.templates == nil {
		return "", nil
	}

	var buf bytesBuffer
	err := t.templates.ExecuteTemplate(&buf, name, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// bytesBuffer - простой буфер для рендеринга
type bytesBuffer struct {
	data []byte
}

func (b *bytesBuffer) Write(p []byte) (n int, err error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *bytesBuffer) String() string {
	return string(b.data)
}

// TemplateMiddleware создаёт middleware для авто-рендеринга
func (t *Template) TemplateMiddleware() Middleware {
	return func(c *Context, next ultrahttp.RouteHandler) {
		// Добавляем шаблонизатор в контекст (через GetUltra)
		next(c.ultra)
	}
}

// HTML рендерит HTML шаблон
func (c *Context) HTML(status int, name string, data interface{}) {
	// Эта функция будет использовать глобальный шаблонизатор
	// или тот, который установлен через SetTemplateRenderer
	if templateRenderer != nil {
		html, err := templateRenderer.RenderToString(name, data)
		if err != nil {
			c.JsonStatus(500, M{
				"error": "Template render error: " + err.Error(),
			})
			return
		}
		c.ultra.SetStatus(status, "")
		c.ultra.SetHTML(html)
		return
	}

	c.JsonStatus(500, M{
		"error": "Template renderer not initialized. Use SetTemplateRenderer()",
	})
}

// глобальный шаблонизатор
var templateRenderer *Template

// SetTemplateRenderer устанавливает глобальный шаблонизатор
func SetTemplateRenderer(t *Template) {
	templateRenderer = t
}

// GetTemplateRenderer возвращает глобальный шаблонизатор
func GetTemplateRenderer() *Template {
	return templateRenderer
}
