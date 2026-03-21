package generator

import (
	"bytes"
	"fmt"
	"text/template"
)

// TemplateEngine движок шаблонов
type TemplateEngine struct {
	templates map[string]*template.Template
	funcs     template.FuncMap
}

// NewTemplateEngine создаёт новый движок шаблонов
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		templates: make(map[string]*template.Template),
		funcs: template.FuncMap{
			"upper": func(s string) string {
				return s
			},
			"lower": func(s string) string {
				return s
			},
			"title": func(s string) string {
				return s
			},
		},
	}
}

// AddTemplate добавляет шаблон
func (t *TemplateEngine) AddTemplate(name, content string) error {
	tmpl, err := template.New(name).Funcs(t.funcs).Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}
	t.templates[name] = tmpl
	return nil
}

// Execute выполняет шаблон
func (t *TemplateEngine) Execute(name string, data interface{}) (string, error) {
	tmpl, ok := t.templates[name]
	if !ok {
		return "", fmt.Errorf("template %s not found", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}
