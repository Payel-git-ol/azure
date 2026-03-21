// Package generator - генератор проектов Azure на основе YAML конфигурации
package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProjectConfig основная структура конфигурации проекта
type ProjectConfig struct {
	MainFile *MainFileConfig           `yaml:"main-file"`
	Project  *ProjectStructure         `yaml:"project"`
	Services map[string]*ServiceConfig `yaml:"services"`
	Database *DatabaseConfig           `yaml:"database,omitempty"`
	Run      interface{}               `yaml:"run"`
}

// MainFileConfig конфигурация главного файла
type MainFileConfig struct {
	Files []string `yaml:",flow"`
}

// ProjectStructure структура проекта
type ProjectStructure struct {
	Internal *InternalStructure `yaml:"internal,omitempty"`
}

// InternalStructure внутренняя структура
type InternalStructure struct {
	Core    *CoreStructure `yaml:"core,omitempty"`
	Fetcher []string       `yaml:"fetcher,omitempty"`
}

// CoreStructure ядро проекта
type CoreStructure struct {
	Services []string `yaml:"services,omitempty"`
}

// ModelConfig конфигурация модели
type ModelConfig struct {
	Path string `yaml:"path"`
	Type string `yaml:"type"`
}

// ServiceConfig конфигурация сервиса
type ServiceConfig struct {
	Imports       []string                   `yaml:"import,omitempty"`
	Azure         *AzureConfig               `yaml:"azure,omitempty"`
	Models        []ModelConfig              `yaml:"models,omitempty"`
	Functions     map[string]*FunctionConfig `yaml:"functions,omitempty"`
	AzureUse      []string                   `yaml:"azure.use,omitempty"`
	AzureHandlers []string                   `yaml:"azure.handlers,omitempty"`
}

// AzureConfig конфигурация Azure
type AzureConfig struct {
	Use      []string `yaml:"use,omitempty"`
	Handlers []string `yaml:"handlers,omitempty"`
}

// FunctionConfig конфигурация функции
type FunctionConfig struct {
	Parameters map[string]string `yaml:"parameters,omitempty"`
	Model      string            `yaml:"model,omitempty"`
	Returns    *ReturnsConfig    `yaml:"returns,omitempty"`
}

// ReturnsConfig конфигурация возврата
type ReturnsConfig struct {
	Model string `yaml:"model,omitempty"`
}

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	Type        string   `yaml:"type,omitempty"`
	AutoMigrate bool     `yaml:"auto-migrate,omitempty"`
	Models      []string `yaml:"models,omitempty"`
}

// Generator генератор проектов
type Generator struct {
	config    *ProjectConfig
	rootDir   string
	templates *TemplateEngine
}

// NewGenerator создаёт новый генератор
func NewGenerator(configPath string) (*Generator, error) {
	// Читаем YAML файл
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Парсим конфигурацию
	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Определяем корневую директорию
	rootDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	return &Generator{
		config:    &config,
		rootDir:   rootDir,
		templates: NewTemplateEngine(),
	}, nil
}

// Generate генерирует проект
func (g *Generator) Generate() error {
	fmt.Println("🚀 Starting project generation...")

	// Создаём структуру директорий
	if err := g.createDirectoryStructure(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Генерируем main.go
	if err := g.generateMain(); err != nil {
		return fmt.Errorf("failed to generate main: %w", err)
	}

	// Генерируем сервисы
	if err := g.generateServices(); err != nil {
		return fmt.Errorf("failed to generate services: %w", err)
	}

	// Генерируем модели
	if err := g.generateModels(); err != nil {
		return fmt.Errorf("failed to generate models: %w", err)
	}

	// Генерируем функции
	if err := g.generateFunctions(); err != nil {
		return fmt.Errorf("failed to generate functions: %w", err)
	}

	// Создаём go.mod
	if err := g.generateGoMod(); err != nil {
		return fmt.Errorf("failed to generate go.mod: %w", err)
	}

	fmt.Println("✅ Project generation completed!")
	return nil
}

// createDirectoryStructure создаёт структуру директорий
func (g *Generator) createDirectoryStructure() error {
	dirs := []string{
		"cmd/app",
		"internal/core/services",
		"internal/fetcher",
		"pkg/models",
		"pkg/requests",
		"handlers",
		"middleware",
		"config",
		"migrations",
	}

	for _, dir := range dirs {
		path := filepath.Join(g.rootDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Printf("📁 Created directory: %s\n", dir)
	}

	return nil
}

// generateMain генерирует main.go
func (g *Generator) generateMain() error {
	var sb strings.Builder

	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"github.com/Payel-git-ol/azure\"\n")
	sb.WriteString("\t\"log\"\n")
	sb.WriteString(")\n\n")
	sb.WriteString("func main() {\n")
	sb.WriteString("\ta := azure.New()\n\n")

	// Добавляем middleware из azure.use
	if g.config.Services != nil {
		if mainSvc, ok := g.config.Services["main"]; ok {
			for _, use := range mainSvc.AzureUse {
				sb.WriteString(fmt.Sprintf("\ta.Use(azure.%s)\n", use))
			}
			sb.WriteString("\n")

			// Генерируем handlers
			for _, handler := range mainSvc.AzureHandlers {
				g.generateHandler(&sb, handler)
			}
		}
	}

	// Добавляем запуск сервера
	port := "7070"
	if g.config.Run != nil {
		switch v := g.config.Run.(type) {
		case string:
			port = v
		case int:
			port = fmt.Sprintf("%d", v)
		}
	}

	sb.WriteString("\n\tlog.Printf(\"🚀 Azure server starting on :%s\")\n")
	sb.WriteString(fmt.Sprintf("\tif err := a.Run(\":%s\"); err != nil {\n", port))
	sb.WriteString("\t\tlog.Fatalf(\"❌ Server error: %v\", err)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n")

	// Записываем файл
	mainPath := filepath.Join(g.rootDir, "cmd/app/main.go")
	if err := os.WriteFile(mainPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write main.go: %w", err)
	}

	fmt.Println("📄 Generated: cmd/app/main.go")
	return nil
}

// generateHandler генерирует обработчик из строки
func (g *Generator) generateHandler(sb *strings.Builder, handler string) {
	// Парсим строку вида: get_root, post_create и т.д.
	switch handler {
	case "get_root":
		sb.WriteString("\n\t// GET / handler\n")
		sb.WriteString("\ta.Get(\"/\", func(c *azure.Context) {\n")
		sb.WriteString("\t\tc.Json(azure.M{\"hello\": \"world\"})\n")
		sb.WriteString("\t})\n")

	case "post_create":
		sb.WriteString("\n\t// POST /post handler with operations\n")
		sb.WriteString("\ta.Post(\"/post\", func(c *azure.Context) {\n")
		sb.WriteString("\t\t// bind(User)\n")
		sb.WriteString("\t\tvar user User\n")
		sb.WriteString("\t\tif err := c.BindJSON(&user); err != nil {\n")
		sb.WriteString("\t\t\tc.JsonStatus(400, azure.M{\"error\": err.Error()})\n")
		sb.WriteString("\t\t\treturn\n")
		sb.WriteString("\t\t}\n\n")
		sb.WriteString("\t\t// operations\n")
		sb.WriteString("\t\t// TODO: Implement your logic here\n\n")
		sb.WriteString("\t\t// json(\"result\" -> \"success\")\n")
		sb.WriteString("\t\tc.Json(azure.M{\"result\": \"success\"})\n")
		sb.WriteString("\t})\n")
	}
}

// generateServices генерирует сервисы
func (g *Generator) generateServices() error {
	if g.config.Project != nil && g.config.Project.Internal != nil {
		if g.config.Project.Internal.Core != nil {
			for _, service := range g.config.Project.Internal.Core.Services {
				serviceName := strings.TrimSuffix(service, ".go")
				servicePath := filepath.Join(g.rootDir, "internal/core/services", service)

				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("package services\n\n"))
				sb.WriteString(fmt.Sprintf("// %s - %s service\n", serviceName, serviceName))
				sb.WriteString("\n// TODO: Implement service logic\n")

				if err := os.WriteFile(servicePath, []byte(sb.String()), 0644); err != nil {
					return fmt.Errorf("failed to write service %s: %w", service, err)
				}
				fmt.Printf("📄 Generated: internal/core/services/%s\n", service)
			}
		}
	}

	return nil
}

// generateModels генерирует модели
func (g *Generator) generateModels() error {
	// Создаём placeholder модели
	userModelPath := filepath.Join(g.rootDir, "pkg/models/user.go")
	userModel := `package models

// User represents a user in the system
type User struct {
	ID       int    ` + "`" + `json:"id"` + "`" + `
	Name     string ` + "`" + `json:"name"` + "`" + `
	Email    string ` + "`" + `json:"email"` + "`" + `
	Age      int    ` + "`" + `json:"age"` + "`" + `
}
`
	if err := os.WriteFile(userModelPath, []byte(userModel), 0644); err != nil {
		return fmt.Errorf("failed to write user model: %w", err)
	}
	fmt.Println("📄 Generated: pkg/models/user.go")

	// Создаём requests модель
	reqModelPath := filepath.Join(g.rootDir, "pkg/requests/user.go")
	reqModel := `package requests

// ReqUser represents a user creation/update request
type ReqUser struct {
	Name  string ` + "`" + `json:"name" validate:"required"` + "`" + `
	Email string ` + "`" + `json:"email" validate:"required,email"` + "`" + `
	Age   int    ` + "`" + `json:"age" validate:"required,min=18"` + "`" + `
}
`
	if err := os.WriteFile(reqModelPath, []byte(reqModel), 0644); err != nil {
		return fmt.Errorf("failed to write request model: %w", err)
	}
	fmt.Println("📄 Generated: pkg/requests/user.go")

	return nil
}

// generateFunctions генерирует функции
func (g *Generator) generateFunctions() error {
	if g.config.Services != nil {
		for serviceName, service := range g.config.Services {
			if len(service.Functions) > 0 {
				funcPath := filepath.Join(g.rootDir, "internal/core/services", serviceName+".go")

				var sb strings.Builder
				sb.WriteString("package services\n\n")

				// Добавляем импорты
				if len(service.Models) > 0 {
					sb.WriteString("import (\n")
					sb.WriteString("\t\"your-project/pkg/models\"\n")
					sb.WriteString(")\n\n")
				}

				// Генерируем функции
				for funcName, funcConfig := range service.Functions {
					sb.WriteString(fmt.Sprintf("// %s - generated function\n", funcName))
					sb.WriteString(fmt.Sprintf("func %s(", funcName))

					// Параметры
					params := make([]string, 0)
					for paramName, paramType := range funcConfig.Parameters {
						params = append(params, fmt.Sprintf("%s %s", paramName, paramType))
					}
					sb.WriteString(strings.Join(params, ", "))

					// Возвращаемые значения
					if funcConfig.Model != "" || (funcConfig.Returns != nil && funcConfig.Returns.Model != "") {
						sb.WriteString(") (*models.User, error) {\n")
					} else {
						sb.WriteString(") {\n")
					}

					sb.WriteString("\t// TODO: Implement logic\n")
					if funcConfig.Model != "" || (funcConfig.Returns != nil && funcConfig.Returns.Model != "") {
						sb.WriteString("\treturn nil, nil\n")
					}
					sb.WriteString("}\n\n")
				}

				if err := os.WriteFile(funcPath, []byte(sb.String()), 0644); err != nil {
					return fmt.Errorf("failed to write functions for %s: %w", serviceName, err)
				}
				fmt.Printf("📄 Generated: internal/core/services/%s.go\n", serviceName)
			}
		}
	}

	return nil
}

// generateGoMod генерирует go.mod
func (g *Generator) generateGoMod() error {
	goMod := `module your-project

go 1.25.4

require (
	github.com/Payel-git-ol/azure v0.1.4
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/Payel-git-ol/azure => ./azure
`

	goModPath := filepath.Join(g.rootDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte(goMod), 0644); err != nil {
		return fmt.Errorf("failed to write go.mod: %w", err)
	}
	fmt.Println("📄 Generated: go.mod")

	return nil
}
