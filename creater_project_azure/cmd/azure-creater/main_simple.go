package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// loadConfig загружает конфигурацию с поддержкой импортов
func loadConfig(configPath string) (*ProjectConfig, error) {
	// Читаем основной файл
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// Получаем директорию конфига для относительных путей
	configDir := filepath.Dir(configPath)

	// Парсим основную конфигурацию
	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Обрабатываем импорты
	if len(config.Import) > 0 {
		for i := range config.Import {
			importFile := config.Import[i].File
			if importFile == "" {
				continue
			}

			// Путь к импортируемому файлу
			importPath := filepath.Join(configDir, importFile)
			importData, err := os.ReadFile(importPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read import %s: %w", importFile, err)
			}

			// Парсим импортированную конфигурацию
			var importConfig ProjectConfig
			if err := yaml.Unmarshal(importData, &importConfig); err != nil {
				return nil, fmt.Errorf("failed to parse import %s: %w", importFile, err)
			}

			// Сливаем экспортированные данные
			if importConfig.Export != nil {
				// Сливаем project
				if importConfig.Export.Project != nil {
					mergeProjectStructure(&config, importConfig.Export.Project)
				}

				// Сливаем services с учетом appeal
				for svcName, svcConfig := range importConfig.Export.Services {
					targetName := svcName
					if config.Import[i].Appeal != "" {
						targetName = config.Import[i].Appeal
					}
					config.Services[targetName] = svcConfig
				}
			}
		}
	}

	return &config, nil
}

// mergeProjectStructure сливает ProjectStructure
func mergeProjectStructure(config *ProjectConfig, exported *ProjectStructure) {
	if exported == nil || config.Project == nil {
		return
	}

	if config.Project.Internal == nil {
		config.Project.Internal = &InternalStructure{}
	}

	if exported.Internal == nil {
		return
	}

	// Сливаем core
	if exported.Internal.Core != nil {
		if config.Project.Internal.Core == nil {
			config.Project.Internal.Core = &CoreStructure{}
		}
		config.Project.Internal.Core.Services = append(
			config.Project.Internal.Core.Services,
			exported.Internal.Core.Services...,
		)
	}

	// Сливаем fetcher
	config.Project.Internal.Fetcher = append(
		config.Project.Internal.Fetcher,
		exported.Internal.Fetcher...,
	)

	// Сливаем pkg
	if exported.Internal.Pkg != nil {
		if config.Project.Internal.Pkg == nil {
			config.Project.Internal.Pkg = &PkgStructure{}
		}
		config.Project.Internal.Pkg.Database = append(
			config.Project.Internal.Pkg.Database,
			exported.Internal.Pkg.Database...,
		)
	}
}

// ImportConfig конфигурация импорта
type ImportConfig struct {
	File   string `yaml:"file"`   // Имя файла
	Appeal string `yaml:"appeal"` // Алиас для импорта
}

// ExportConfig конфигурация экспорта
type ExportConfig struct {
	Project  *ProjectStructure         `yaml:"project,omitempty"`
	Services map[string]*ServiceConfig `yaml:"services,omitempty"`
}

// ProjectConfig полная конфигурация проекта
type ProjectConfig struct {
	Import   []ImportConfig            `yaml:"import,omitempty"`
	Export   *ExportConfig             `yaml:"export,omitempty"`
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
	Pkg     *PkgStructure  `yaml:"pkg"`
}

// PkgStructure структура pkg
type PkgStructure struct {
	Database []string `yaml:"database"`
}

// DatabaseConfig конфигурация базы данных
type DatabaseConfig struct {
	ORM         string             `yaml:"orm"`
	Driver      string             `yaml:"driver"`
	DNS         interface{}        `yaml:"dns"`
	AutoMigrate *AutoMigrateConfig `yaml:"auto-migrate"`
}

// AutoMigrateConfig конфигурация auto-migrate
type AutoMigrateConfig struct {
	Models []string `yaml:"model"`
}

// CoreStructure ядро проекта
type CoreStructure struct {
	Services []string `yaml:"services"`
}

// ServiceConfig конфигурация сервиса
type ServiceConfig struct {
	Type          string                     `yaml:"type"`
	AzureUse      []string                   `yaml:"azure.use"`
	AzureHandlers interface{}                `yaml:"azure.handlers"`
	Models        []ModelConfig              `yaml:"models"`
	Functions     map[string]*FunctionConfig `yaml:"functions"`
	Run           interface{}                `yaml:"run"`
	Database      *DatabaseConfig            `yaml:"database"`
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
	Path   string        `yaml:"path"`
	Type   string        `yaml:"type"`   // model или request
	Name   string        `yaml:"name"`   // Имя структуры
	Fields []FieldConfig `yaml:"fields"` // Поля
}

// FieldConfig конфигурация поля
type FieldConfig struct {
	Name string `yaml:"name"`
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

	// Загружаем конфигурацию с поддержкой импортов
	config, err := loadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ Error loading config: %v\n", err)
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

		// Генерируем pkg/database
		if config.Project.Internal.Pkg != nil && len(config.Project.Internal.Pkg.Database) > 0 {
			dirs["pkg/database"] = true
			dbDir := filepath.Join(rootDir, "pkg/database")
			if err := os.MkdirAll(dbDir, 0755); err != nil {
				fmt.Printf("❌ Error creating database directory: %v\n", err)
				os.Exit(1)
			}

			// Создаём database.go
			dbPath := filepath.Join(dbDir, "database.go")

			// Проверяем тип DNS
			dnsVarName := "DATABASE_DNS"
			useAzureEnv := false

			if config.Services != nil {
				if dbSvc, ok := config.Services["database"]; ok && dbSvc.Database != nil && dbSvc.Database.DNS != nil {
					// Проверяем azure.env
					if dnsMap, ok := dbSvc.Database.DNS.(map[string]interface{}); ok {
						if azureEnv, ok := dnsMap["azure.env"]; ok {
							if envName, ok := azureEnv.(string); ok {
								dnsVarName = envName
								useAzureEnv = true
							}
						} else if env, ok := dnsMap["env"]; ok {
							// Простой env - используем os.Getenv
							if envName, ok := env.(string); ok {
								dnsVarName = envName
								useAzureEnv = false
							}
						}
					}
				}
			}

			var dbContent string
			if useAzureEnv {
				dbContent = fmt.Sprintf(`package database

import (
	"log"

	"github.com/Payel-git-ol/azure/env"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB инициализирует базу данных и выполняет миграцию
func InitDB(models ...interface{}) {
	dsn := env.MustGet("%s", "")
	var err error
	
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Database error: %%v", err)
	}

	if err = DB.AutoMigrate(models...); err != nil {
		log.Fatalf("❌ Migration error: %%v", err)
	}
	
	log.Println("✅ Database initialized and migrated")
}
`, dnsVarName)
			} else {
				dbContent = `package database

import (
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB инициализирует базу данных и выполняет миграцию
func InitDB(models ...interface{}) {
	dsn := os.Getenv("DATABASE_DNS")
	var err error
	
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("❌ Database error: %v", err)
	}

	if err = DB.AutoMigrate(models...); err != nil {
		log.Fatalf("❌ Migration error: %v", err)
	}
	
	log.Println("✅ Database initialized and migrated")
}
`
			}

			if err := os.WriteFile(dbPath, []byte(dbContent), 0644); err != nil {
				fmt.Printf("❌ Error creating database.go: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("📄 Generated: pkg/database/database.go\n")
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

	// Проверяем есть ли database сервис
	hasDatabase := false
	if config.Services != nil {
		if dbSvc, ok := config.Services["database"]; ok && dbSvc.Database != nil {
			hasDatabase = true
		}
	}

	mainContent := generateMain(config, hasDatabase)
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		fmt.Printf("❌ Error writing main.go: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("📄 Generated: cmd/app/main.go")

	// Генерируем модели
	if config.Services != nil {
		for svcName, svc := range config.Services {
			for _, model := range svc.Models {
				// Определяем имя пакета из пути
				pkgName := getPackageName(model.Path)

				// Определяем тип модели
				modelType := model.Type
				if modelType == "" {
					modelType = "model" // по умолчанию
				}

				// Генерируем контент модели
				var modelContent strings.Builder
				modelContent.WriteString(fmt.Sprintf("package %s\n\n", pkgName))

				// Имя структуры
				structName := model.Name
				if structName == "" {
					// Берём из имени файла
					structName = strings.TrimSuffix(filepath.Base(model.Path), ".go")
					structName = strings.Title(structName)
				}

				modelContent.WriteString(fmt.Sprintf("// %s - generated %s\n", structName, modelType))
				modelContent.WriteString(fmt.Sprintf("type %s struct {\n", structName))

				// Генерируем поля
				if len(model.Fields) > 0 {
					for _, field := range model.Fields {
						fieldName := strings.Title(field.Name)
						fieldType := field.Type
						if fieldType == "" {
							fieldType = "string"
						}

						if modelType == "request" {
							// Request с json тегами
							modelContent.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, field.Name))
						} else {
							// Model без тегов (или с json тегами по умолчанию)
							modelContent.WriteString(fmt.Sprintf("\t%s %s `json:\"%s\"`\n", fieldName, fieldType, field.Name))
						}
					}
				} else {
					// Пустая структура если нет полей
					modelContent.WriteString("\t// TODO: Add fields\n")
				}

				modelContent.WriteString("}\n")

				// Записываем файл
				modelPath := filepath.Join(rootDir, model.Path)
				if err := os.WriteFile(modelPath, []byte(modelContent.String()), 0644); err != nil {
					fmt.Printf("❌ Error writing model %s: %v\n", model.Path, err)
					os.Exit(1)
				}
				fmt.Printf("📄 Generated: %s\n", model.Path)
			}

			// Не генерируем модели для svcName == "user" здесь, чтобы избежать дублирования
			if svcName == "user" {
				continue
			}
		}
	}

	// Генерируем сервисы
	if config.Project != nil && config.Project.Internal != nil && config.Project.Internal.Core != nil {
		// Генерируем сервисы для всех сервисов в config.Services
		for svcName, svcConfig := range config.Services {
			// Пропускаем main и database - они не сервисы
			if svcName == "main" || svcName == "database" {
				continue
			}

			// Имя файла сервиса
			serviceFile := svcName + ".go"
			servicePath := filepath.Join(rootDir, "internal/core/services", serviceFile)

			var serviceContent strings.Builder

			// Собираем импорты для моделей
			imports := make(map[string]string) // pkg -> path
			for _, model := range svcConfig.Models {
				pkgName := getPackageName(model.Path)
				importPath := getModelImportPath(model.Path)
				imports[pkgName] = importPath
			}

			// Записываем package и импорты
			serviceContent.WriteString("package services\n\n")

			// Собираем все импорты
			allImports := make(map[string]bool)

			// Добавляем импорты моделей
			for _, model := range svcConfig.Models {
				importPath := getModelImportPath(model.Path)
				allImports[importPath] = true
			}

			// Проверяем code на наличие time.Time
			for _, funcConfig := range svcConfig.Functions {
				if funcConfig.Code != nil {
					if codeStr, ok := funcConfig.Code.(string); ok {
						if strings.Contains(codeStr, "time.") {
							allImports["time"] = true
						}
					}
				}
				// Проверяем параметры на time.Time
				for _, paramType := range funcConfig.Parameters {
					if paramType == "time.Time" {
						allImports["time"] = true
					}
				}
			}

			// Всегда добавляем log
			allImports["log"] = true

			// Записываем импорты
			if len(allImports) > 0 {
				serviceContent.WriteString("import (\n")
				for importPath := range allImports {
					serviceContent.WriteString(fmt.Sprintf("\t\"%s\"\n", importPath))
				}
				serviceContent.WriteString(")\n\n")
			}

			// Генерируем функции
			for funcName, funcConfig := range svcConfig.Functions {
				serviceContent.WriteString(fmt.Sprintf("// %s - generated function\n", funcName))
				serviceContent.WriteString(fmt.Sprintf("func %s(", funcName))

				params := make([]string, 0)
				for paramName, paramType := range funcConfig.Parameters {
					params = append(params, fmt.Sprintf("%s %s", paramName, paramType))
				}
				serviceContent.WriteString(strings.Join(params, ", "))

				if funcConfig.Model != "" {
					// Находим путь к модели
					var modelPath string
					for _, model := range svcConfig.Models {
						if model.Name == funcConfig.Model {
							modelPath = model.Path
							break
						}
					}

					if modelPath != "" {
						modelRef := getModelRef(modelPath, funcConfig.Model)
						serviceContent.WriteString(fmt.Sprintf(") %s {\n", modelRef))
					} else {
						serviceContent.WriteString(fmt.Sprintf(") (*models.%s, error) {\n", funcConfig.Model))
					}

					// Пользовательский код из YAML (code поле)
					hasReturn := false
					if funcConfig.Code != nil {
						if codeStr, ok := funcConfig.Code.(string); ok && codeStr != "" {
							lines := strings.Split(codeStr, "\n")
							for _, line := range lines {
								line = strings.TrimSpace(line)
								if line != "" {
									serviceContent.WriteString(fmt.Sprintf("\t%s\n", line))
									if strings.HasPrefix(line, "return") {
										hasReturn = true
									}
								}
							}
						} else if codeLines, ok := funcConfig.Code.([]interface{}); ok {
							for _, line := range codeLines {
								if lineStr, ok := line.(string); ok && lineStr != "" {
									serviceContent.WriteString(fmt.Sprintf("\t%s\n", lineStr))
									if strings.HasPrefix(lineStr, "return") {
										hasReturn = true
									}
								}
							}
						}
					} else {
						serviceContent.WriteString("\t// TODO: Implement logic\n")
					}

					// Добавляем return только если его нет в коде
					if !hasReturn {
						serviceContent.WriteString("\treturn nil, nil\n")
					}
				} else {
					serviceContent.WriteString(") {\n")

					// Пользовательский код из YAML (code поле)
					if funcConfig.Code != nil {
						if codeStr, ok := funcConfig.Code.(string); ok && codeStr != "" {
							lines := strings.Split(codeStr, "\n")
							for _, line := range lines {
								line = strings.TrimSpace(line)
								if line != "" && !strings.HasPrefix(line, "return") {
									serviceContent.WriteString(fmt.Sprintf("\t%s\n", line))
								}
							}
						} else if codeLines, ok := funcConfig.Code.([]interface{}); ok {
							for _, line := range codeLines {
								if lineStr, ok := line.(string); ok && lineStr != "" && !strings.HasPrefix(lineStr, "return") {
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

			// Записываем файл только если есть функции
			if len(svcConfig.Functions) > 0 {
				if err := os.WriteFile(servicePath, []byte(serviceContent.String()), 0644); err != nil {
					fmt.Printf("❌ Error writing service %s: %v\n", serviceFile, err)
					os.Exit(1)
				}
				fmt.Printf("📄 Generated: internal/core/services/%s\n", serviceFile)
			}
		}
	}

	// Создаём go.mod
	goModPath := filepath.Join(rootDir, "go.mod")
	goMod := `module your-project

go 1.25.4

require (
	github.com/Payel-git-ol/azure v0.1.4
	github.com/Payel-git-ol/azure/env v0.0.0
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/gorm v1.25.0
	gorm.io/driver/postgres v1.5.0
)

replace github.com/Payel-git-ol/azure => ./azure

replace github.com/Payel-git-ol/azure/env => ./azure/env
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
func generateMain(config *ProjectConfig, hasDatabase bool) string {
	var sb strings.Builder

	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n")
	if hasDatabase {
		sb.WriteString("\t\"os\"\n")
		sb.WriteString("\t\"your-project/pkg/database\"\n")
		sb.WriteString("\t\"your-project/pkg/models\"\n")
	}
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

	// Добавляем инициализацию БД
	if hasDatabase {
		sb.WriteString("\n\t// Инициализация базы данных\n")
		sb.WriteString("\tdatabase.InitDB(&models.User{})\n\n")
	}

	// Добавляем запуск сервера
	port := "7070"
	if config.Services != nil {
		if mainSvc, ok := config.Services["main"]; ok && mainSvc.Run != nil {
			switch v := mainSvc.Run.(type) {
			case string:
				port = v
			case int:
				port = fmt.Sprintf("%d", v)
			}
		}
	}

	sb.WriteString(fmt.Sprintf("\tlog.Println(\"🚀 Azure server starting on :%s\")\n", port))
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

// getPackageName извлекает имя пакета из пути
func getPackageName(path string) string {
	dir := filepath.Dir(path)
	// Заменяем обратные слеши на прямые
	dir = filepath.ToSlash(dir)
	// Используем только последнюю часть пути (имя папки)
	parts := strings.Split(dir, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "models"
}

// getModelImportPath возвращает полный путь импорта для модели
func getModelImportPath(path string) string {
	dir := filepath.Dir(path)
	// Заменяем обратные слеши на прямые
	dir = filepath.ToSlash(dir)
	// Для путей вида pkg/models или pkg/requests
	if strings.HasPrefix(dir, "pkg/") {
		return "your-project/" + dir
	}
	return "your-project/" + dir
}

// getModelRef возвращает ссылку на модель с префиксом пакета
func getModelRef(path string, modelName string) string {
	pkgName := getPackageName(path)
	return fmt.Sprintf("*%s.%s", pkgName, modelName)
}
