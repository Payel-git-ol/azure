package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProjectConfig полная конфигурация проекта
type ProjectConfig struct {
	MainFile *MainFileConfig           `yaml:"main-file"`
	Project  *ProjectStructure         `yaml:"project"`
	Services map[string]*ServiceConfig `yaml:"services"`
	Run      interface{}               `yaml:"run"`
}

// MainFileConfig конфигурация главного файла
type MainFileConfig struct {
	Files []string `yaml:"files"`
}

// ProjectStructure структура проекта
type ProjectStructure struct {
	Internal *InternalStructure `yaml:"internal"`
}

// InternalStructure внутренняя структура
type InternalStructure struct {
	Core    *CoreStructure `yaml:"core"`
	Fetcher []string       `yaml:"fetcher"`
}

// CoreStructure ядро проекта
type CoreStructure struct {
	Services []string `yaml:"services"`
}

// ServiceConfig конфигурация сервиса
type ServiceConfig struct {
	AzureUse      []string                   `yaml:"azure.use"`
	AzureHandlers interface{}                `yaml:"azure.handlers"` // Может быть []string или map[string]*HandlerConfig
	Models        []ModelConfig              `yaml:"models"`
	Functions     map[string]*FunctionConfig `yaml:"functions"`
}

// HandlerConfig конфигурация хендлера (новый формат)
type HandlerConfig struct {
	Method     string        `yaml:"method"`
	Operations interface{}   `yaml:"operations,omitempty"`
	Result     *ResultConfig `yaml:"result"`
}

// ResultConfig конфигурация результата
type ResultConfig struct {
	JSON interface{} `yaml:"json"`
}

// ModelConfig конфигурация модели
type ModelConfig struct {
	Path string `yaml:"path"`
	Type string `yaml:"type"`
}

// FunctionConfig конфигурация функции
type FunctionConfig struct {
	Parameters map[string]string `yaml:"parameters"`
	Model      string            `yaml:"model"`
	Code       interface{}       `yaml:"code"` // Пользовательский код
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: azure-creater <config.yaml>")
		fmt.Println("Example: azure-creater create-api-azure.yaml")
		os.Exit(1)
	}

	configPath := os.Args[1]

	// Читаем YAML
	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("❌ Error reading config: %v\n", err)
		os.Exit(1)
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Printf("❌ Error parsing config: %v\n", err)
		os.Exit(1)
	}

	// Получаем корневую директорию
	rootDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("❌ Error getting working directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🔧 Azure Project Creator")
	fmt.Println("========================")
	fmt.Printf("📄 Config: %s\n\n", configPath)
	fmt.Println("🚀 Starting project generation...")

	// Создаём директории ТОЛЬКО из YAML
	dirs := make(map[string]bool)

	// Добавляем main-file directory
	if config.MainFile != nil && len(config.MainFile.Files) > 0 {
		for _, file := range config.MainFile.Files {
			dir := filepath.Dir(file)
			dirs[dir] = true
		}
	}

	// Добавляем project structure
	if config.Project != nil && config.Project.Internal != nil {
		if config.Project.Internal.Core != nil {
			dirs["internal/core"] = true
			dirs["internal/core/services"] = true
		}
		if len(config.Project.Internal.Fetcher) > 0 {
			dirs["internal/fetcher"] = true
			// Создаём директорию
			fetcherDir := filepath.Join(rootDir, "internal/fetcher")
			if err := os.MkdirAll(fetcherDir, 0755); err != nil {
				fmt.Printf("❌ Error creating fetcher directory: %v\n", err)
				os.Exit(1)
			}
			// Создаём файлы
			for _, fetcherFile := range config.Project.Internal.Fetcher {
				fetcherPath := filepath.Join(fetcherDir, fetcherFile)
				fetcherContent := fmt.Sprintf(`package fetcher

// %s - generated fetcher
// TODO: Implement fetcher logic
`, strings.TrimSuffix(fetcherFile, ".go"))
				if err := os.WriteFile(fetcherPath, []byte(fetcherContent), 0644); err != nil {
					fmt.Printf("❌ Error creating fetcher %s: %v\n", fetcherFile, err)
					os.Exit(1)
				}
				fmt.Printf("📄 Generated: internal/fetcher/%s\n", fetcherFile)
			}
		}
	}

	// Добавляем модели из сервисов
	if config.Services != nil {
		for _, svc := range config.Services {
			for _, model := range svc.Models {
				dir := filepath.Dir(model.Path)
				dirs[dir] = true
			}
		}
	}

	// Создаём все директории
	for dir := range dirs {
		path := filepath.Join(rootDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			fmt.Printf("❌ Error creating directory %s: %v\n", dir, err)
			os.Exit(1)
		}
		fmt.Printf("📁 Created directory: %s\n", dir)
	}

	// Генерируем main.go
	mainPath := filepath.Join(rootDir, "cmd/app/main.go")
	mainContent := generateMain(&config)
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		fmt.Printf("❌ Error writing main.go: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("📄 Generated: cmd/app/main.go")

	// Генерируем модели
	if config.Services != nil {
		if userSvc, ok := config.Services["user"]; ok {
			for _, model := range userSvc.Models {
				var modelContent string
				if strings.Contains(model.Path, "requests") {
					modelContent = `package requests

// ReqUser represents a user creation/update request
type ReqUser struct {
	Name  string ` + "`" + `json:"name" validate:"required"` + "`" + `
	Email string ` + "`" + `json:"email" validate:"required,email"` + "`" + `
	Age   int    ` + "`" + `json:"age" validate:"required,min=18"` + "`" + `
}
`
				} else {
					modelContent = `package models

// User represents a user in the system
type User struct {
	ID       int    ` + "`" + `json:"id"` + "`" + `
	Name     string ` + "`" + `json:"name"` + "`" + `
	Email    string ` + "`" + `json:"email"` + "`" + `
	Age      int    ` + "`" + `json:"age"` + "`" + `
}
`
				}

				modelPath := filepath.Join(rootDir, model.Path)
				if err := os.WriteFile(modelPath, []byte(modelContent), 0644); err != nil {
					fmt.Printf("❌ Error writing model %s: %v\n", model.Path, err)
					os.Exit(1)
				}
				fmt.Printf("📄 Generated: %s\n", model.Path)
			}
		}
	}

	// Генерируем сервисы
	if config.Project != nil && config.Project.Internal != nil && config.Project.Internal.Core != nil {
		for _, service := range config.Project.Internal.Core.Services {
			servicePath := filepath.Join(rootDir, "internal/core/services", service)

			var serviceContent strings.Builder
			serviceContent.WriteString("package services\n\n")

			// Генерируем функции из user сервиса
			if config.Services != nil {
				if userSvc, ok := config.Services["user"]; ok {
					for funcName, funcConfig := range userSvc.Functions {
						serviceContent.WriteString(fmt.Sprintf("// %s - generated function\n", funcName))
						serviceContent.WriteString(fmt.Sprintf("func %s(", funcName))

						params := make([]string, 0)
						for paramName, paramType := range funcConfig.Parameters {
							params = append(params, fmt.Sprintf("%s %s", paramName, paramType))
						}
						serviceContent.WriteString(strings.Join(params, ", "))

						if funcConfig.Model != "" {
							serviceContent.WriteString(") (*User, error) {\n")

							// Пользовательский код из YAML (code поле)
							if funcConfig.Code != nil {
								if codeStr, ok := funcConfig.Code.(string); ok && codeStr != "" {
									lines := strings.Split(codeStr, ";")
									for _, line := range lines {
										line = strings.TrimSpace(line)
										if line != "" {
											serviceContent.WriteString(fmt.Sprintf("\t%s\n", line))
										}
									}
								} else if codeLines, ok := funcConfig.Code.([]interface{}); ok {
									for _, line := range codeLines {
										if lineStr, ok := line.(string); ok && lineStr != "" {
											serviceContent.WriteString(fmt.Sprintf("\t%s\n", lineStr))
										}
									}
								}
							} else {
								serviceContent.WriteString("\t// TODO: Implement logic\n")
							}

							serviceContent.WriteString("\treturn nil, nil\n")
						} else {
							serviceContent.WriteString(") {\n")

							// Пользовательский код из YAML (code поле)
							if funcConfig.Code != nil {
								if codeStr, ok := funcConfig.Code.(string); ok && codeStr != "" {
									lines := strings.Split(codeStr, ";")
									for _, line := range lines {
										line = strings.TrimSpace(line)
										if line != "" {
											serviceContent.WriteString(fmt.Sprintf("\t%s\n", line))
										}
									}
								} else if codeLines, ok := funcConfig.Code.([]interface{}); ok {
									for _, line := range codeLines {
										if lineStr, ok := line.(string); ok && lineStr != "" {
											serviceContent.WriteString(fmt.Sprintf("\t%s\n", lineStr))
										}
									}
								}
							} else {
								serviceContent.WriteString("\t// TODO: Implement logic\n")
							}
						}
						serviceContent.WriteString("}\n\n")
					}
				}
			}

			if err := os.WriteFile(servicePath, []byte(serviceContent.String()), 0644); err != nil {
				fmt.Printf("❌ Error writing service %s: %v\n", service, err)
				os.Exit(1)
			}
			fmt.Printf("📄 Generated: %s\n", service)
		}
	}

	// Создаём go.mod
	goModPath := filepath.Join(rootDir, "go.mod")
	goMod := `module your-project

go 1.25.4

require (
	github.com/Payel-git-ol/azure v0.1.4
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/Payel-git-ol/azure => ./azure
`
	if err := os.WriteFile(goModPath, []byte(goMod), 0644); err != nil {
		fmt.Printf("❌ Error writing go.mod: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("📄 Generated: go.mod")

	fmt.Println("\n✅ Project generation completed!")
	fmt.Println("\n✨ Next steps:")
	fmt.Println("   1. Review generated files")
	fmt.Println("   2. Implement your logic in // TODO sections")
	fmt.Println("   3. Run: go mod tidy")
	fmt.Println("   4. Run: go run cmd/app/main.go")
}

// generateMain генерирует main.go
func generateMain(config *ProjectConfig) string {
	var sb strings.Builder

	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"github.com/Payel-git-ol/azure\"\n")
	sb.WriteString("\t\"log\"\n")
	sb.WriteString(")\n\n")
	sb.WriteString("func main() {\n")
	sb.WriteString("\ta := azure.New()\n\n")

	// Добавляем middleware
	if config.Services != nil {
		if mainSvc, ok := config.Services["main"]; ok {
			for _, use := range mainSvc.AzureUse {
				sb.WriteString(fmt.Sprintf("\ta.Use(azure.%s)\n", use))
			}
			sb.WriteString("\n")

			// Генерируем handlers
			generateHandlers(&sb, mainSvc.AzureHandlers)
		}
	}

	// Добавляем запуск сервера
	port := "7070"
	if config.Run != nil {
		switch v := config.Run.(type) {
		case string:
			port = v
		case int:
			port = fmt.Sprintf("%d", v)
		}
	}

	sb.WriteString(fmt.Sprintf("\n\tlog.Println(\"🚀 Azure server starting on :%s\")\n", port))
	sb.WriteString(fmt.Sprintf("\tif err := a.Run(\":%s\"); err != nil {\n", port))
	sb.WriteString("\t\tlog.Fatalf(\"❌ Server error: %v\", err)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n")

	return sb.String()
}

// generateHandlers генерирует обработчики
func generateHandlers(sb *strings.Builder, handlers interface{}) {
	switch v := handlers.(type) {
	case []interface{}:
		// Старый формат: ["get_root", "post_create"]
		for _, handler := range v {
			if handlerStr, ok := handler.(string); ok {
				generateNamedHandler(sb, handlerStr)
			}
		}

	case map[string]interface{}:
		// Новый формат: {"/": {method: get, ...}, "/post": {...}}
		for path, handler := range v {
			if handlerMap, ok := handler.(map[string]interface{}); ok {
				generateMapHandler(sb, path, handlerMap)
			}
		}
	}
}

// generateNamedHandler генерирует обработчик по имени
func generateNamedHandler(sb *strings.Builder, name string) {
	switch name {
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
		sb.WriteString("\t\tc.Json(azure.M{\"result\": \"success\"})\n")
		sb.WriteString("\t})\n")
	}
}

// generateMapHandler генерирует обработчик из map
func generateMapHandler(sb *strings.Builder, path string, handler map[string]interface{}) {
	method, _ := handler["method"].(string)
	methodUpper := strings.ToUpper(method)

	sb.WriteString(fmt.Sprintf("\n\t// %s %s handler\n", methodUpper, path))

	// Определяем метод роутера
	var routeMethod string
	switch method {
	case "get":
		routeMethod = "Get"
	case "post":
		routeMethod = "Post"
	case "put":
		routeMethod = "Put"
	case "delete":
		routeMethod = "Delete"
	case "patch":
		routeMethod = "Patch"
	default:
		routeMethod = "Get"
	}

	// Открываем handler
	sb.WriteString(fmt.Sprintf("\ta.%s(\"%s\", func(c *azure.Context) {\n", routeMethod, path))

	// Проверяем operations (bind)
	if ops, ok := handler["operations"]; ok {
		if opsStr, ok := ops.(string); ok && strings.Contains(opsStr, "bind") {
			sb.WriteString("\t\t// bind(User)\n")
			sb.WriteString("\t\tvar user User\n")
			sb.WriteString("\t\tif err := c.BindJSON(&user); err != nil {\n")
			sb.WriteString("\t\t\tc.JsonStatus(400, azure.M{\"error\": err.Error()})\n")
			sb.WriteString("\t\t\treturn\n")
			sb.WriteString("\t\t}\n\n")
		}
	}

	// operations placeholder (если есть operations)
	if _, hasOps := handler["operations"]; hasOps {
		sb.WriteString("\t\t// operations\n")
		sb.WriteString("\t\t// TODO: Implement your logic here\n\n")
	}

	// Пользовательский код из YAML (code поле)
	if code, ok := handler["code"]; ok {
		if codeStr, ok := code.(string); ok && codeStr != "" {
			// Может быть несколько строк кода через ;
			lines := strings.Split(codeStr, ";")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					sb.WriteString(fmt.Sprintf("\t\t%s\n", line))
				}
			}
			sb.WriteString("\n")
		} else if codeLines, ok := code.([]interface{}); ok {
			// Или список строк
			for _, line := range codeLines {
				if lineStr, ok := line.(string); ok && lineStr != "" {
					sb.WriteString(fmt.Sprintf("\t\t%s\n", lineStr))
				}
			}
			sb.WriteString("\n")
		}
	}

	// Результат (если есть)
	if result, ok := handler["result"].(map[string]interface{}); ok {
		if json, ok := result["json"].(map[string]interface{}); ok {
			sb.WriteString("\t\tc.Json(azure.M{")
			parts := make([]string, 0)
			for k, v := range json {
				parts = append(parts, fmt.Sprintf("\"%s\": \"%v\"", k, v))
			}
			sb.WriteString(strings.Join(parts, ", "))
			sb.WriteString("})\n")
		}
	}

	sb.WriteString("\t})\n")
}

// generateConfigHandler генерирует обработчик из HandlerConfig
func generateConfigHandler(sb *strings.Builder, path string, config *HandlerConfig) {
	method := strings.ToUpper(config.Method)
	sb.WriteString(fmt.Sprintf("\n\t// %s %s handler\n", method, path))

	if config.Method == "get" {
		sb.WriteString(fmt.Sprintf("\ta.Get(\"%s\", func(c *azure.Context) {\n", path))
	} else if config.Method == "post" {
		sb.WriteString(fmt.Sprintf("\ta.Post(\"%s\", func(c *azure.Context) {\n", path))

		// Проверяем operations
		if config.Operations != nil {
			if opsStr, ok := config.Operations.(string); ok && strings.Contains(opsStr, "bind") {
				sb.WriteString("\t\t// bind(User)\n")
				sb.WriteString("\t\tvar user User\n")
				sb.WriteString("\t\tif err := c.BindJSON(&user); err != nil {\n")
				sb.WriteString("\t\t\tc.JsonStatus(400, azure.M{\"error\": err.Error()})\n")
				sb.WriteString("\t\t\treturn\n")
				sb.WriteString("\t\t}\n\n")
			}
		}

		sb.WriteString("\t\t// operations\n")
		sb.WriteString("\t\t// TODO: Implement your logic here\n\n")
	}

	// Результат
	if config.Result != nil && config.Result.JSON != nil {
		if json, ok := config.Result.JSON.(map[string]interface{}); ok {
			sb.WriteString("\t\tc.Json(azure.M{")
			parts := make([]string, 0)
			for k, v := range json {
				parts = append(parts, fmt.Sprintf("\"%s\": \"%v\"", k, v))
			}
			sb.WriteString(strings.Join(parts, ", "))
			sb.WriteString("})\n")
		}
	}

	sb.WriteString("\t})\n")
}
