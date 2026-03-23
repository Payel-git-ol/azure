# Azure Project Creator

Генератор проектов на основе YAML конфигурации с поддержкой импортов и модульности.

## 🚀 Быстрый старт

```bash
# Создаём проект
azure-creater create-api-azure.yaml
```

## 📋 Пример использования

### 1. Создаём главный YAML файл

**main.yaml:**
```yaml
import:
  - file: ai-service.yaml
    appeal: ai  # Важно: это имя будет использоваться для сервиса

main-file:
  files:
    - "cmd/app/main.go"

project:
  internal:
    core:
      services:
        - "user.go"
    fetcher:
      - "kafka.go"
    pkg:
      database:
        - "database.go"

services:
  main:
    type: azure.main
    azure.use:
      - Logger()
      - Recovery()
    
    azure.handlers:
      "/":
        method: get
        result:
          json:
            "hello": "world"
    
    run: "7070"

  user:
    type: services
    models:
      - path: "pkg/models/user.go"
        type: model
        name: "User"
        fields:
          - name: id
            type: int
          - name: name
            type: string
    
    functions:
      UserSave:
        parameters:
          name: string
          age: int
        model: User
        code: 'log.Println("save")'

  # ⚠️ Важно: нужно объявить сервис с именем из appeal
  # Это активирует импорт функций из ai-service.yaml
  ai:  # Должно совпадать с appeal в import

  database:
    type: database
    database:
      orm: gorm
      driver: postgres
      dns:
        azure.env: "DATABASE_DNS"
      auto-migrate:
        model:
          - User
```

### 2. Создаём импортируемый файл

**ai-service.yaml:**
```yaml
export:
  project:
    internal:
      fetcher:
        - "ai.go"

  services:
    ai:  # Имя сервиса (будет импортировано в main.yaml как ai)
      functions:
        GetApiKeyHealth:
          parameters:
            key: string
          code: 'log.Println(key)'
```

### 3. Генерируем проект

```bash
azure-creater main.yaml
```

### 4. Получаем структуру

```
your-project/
├── cmd/app/
│   └── main.go              # Сгенерированный main
├── internal/core/services/
│   ├── user.go              # Сервис user
│   └── ai.go                # Сервис ai (из импорта)
├── internal/fetcher/
│   ├── kafka.go
│   └── ai.go
├── pkg/models/
│   └── user.go
├── pkg/requests/
│   └── user.go
├── pkg/database/
│   └── database.go
└── go.mod
```

## 📖 Синтаксис YAML

### **Импорт файлов**

```yaml
import:
  - file: ai-service.yaml    # Имя файла
    appeal: ai               # Алиас (имя сервиса)
```

**⚠️ Важно:** После импорта нужно объявить пустой сервис с именем из `appeal` в главном файле:

```yaml
services:
  ai:  # Должно совпадать с appeal
```

Это активирует импорт функций из импортированного файла.

### **Экспорт из файла**

```yaml
export:
  project:                   # Экспорт структуры проекта
    internal:
      fetcher:
        - "ai.go"
  
  services:                  # Экспорт сервисов
    ai:
      functions:
        MyFunction:
          parameters:
            param: string
          code: 'log.Println("hello")'
```

### **Типы сервисов**

| Тип | Описание |
|-----|----------|
| `azure.main` | Главный сервис Azure |
| `services` | Обычный сервис с функциями |
| `database` | Сервис базы данных |

### **HTTP Handlers**

```yaml
services:
  main:
    azure.handlers:
      "/path":
        method: get|post|put|delete|patch
        operations: "bind(User)"  # Опционально
        code: 'log.Println("custom")'  # Опционально
        result:
          json:
            "key": "value"
```

### **Модели**

```yaml
models:
  - path: "pkg/models/user.go"
    type: model|request        # model или request
    name: "User"               # Имя структуры
    fields:
      - name: id
        type: int
      - name: name
        type: string
```

**Разница между `model` и `request`:**
- `model` - простая структура с JSON тегами
- `request` - структура с JSON тегами для валидации

### **Функции**

```yaml
functions:
  UserSave:
    parameters:
      name: string
      age: int
    model: User              # Возвращаемый тип
    code: 'log.Println("save")'  # Пользовательский код
```

### **База данных**

```yaml
database:
  orm: gorm                  # gorm (пока только он)
  driver: postgres           # postgres, mysql, sqlite
  dns:
    env: "DATABASE_DNS"      # os.Getenv("DATABASE_DNS")
    # или
    azure.env: "DATABASE_DNS"  # env.MustGet("DATABASE_DNS", "")
  auto-migrate:
    model:
      - User                 # Модели для миграции
```

## 🔧 Генерируемый код

### **Из HTTP handler:**

```yaml
"/post":
  method: post
  operations: "bind(User)"
  code: 'log.Println("save")'
  result:
    json:
      "result": "success"
```

**Генерирует:**
```go
a.Post("/post", func(c *azure.Context) {
    // bind(User)
    var user User
    if err := c.BindJSON(&user); err != nil {
        c.JsonStatus(400, azure.M{"error": err.Error()})
        return
    }

    // operations
    // TODO: Implement your logic here

    log.Println("save")

    c.Json(azure.M{"result": "success"})
})
```

### **Из функции:**

```yaml
UserSave:
  parameters:
    name: string
    age: int
  model: User
  code: 'log.Println("save")'
```

**Генерирует:**
```go
func UserSave(name string, age int) *models.User {
    log.Println("save")
    return nil, nil
}
```

### **Из database:**

```yaml
database:
  orm: gorm
  driver: postgres
  dns:
    azure.env: "DATABASE_DNS"
  auto-migrate:
    model:
      - User
```

**Генерирует:**
```go
// pkg/database/database.go
package database

import (
    "log"
    "github.com/Payel-git-ol/azure/env"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

var DB *gorm.DB

func InitDB(models ...interface{}) {
    dsn := env.MustGet("DATABASE_DNS", "")
    DB, _ = gorm.Open(postgres.Open(dsn), &gorm.Config{})
    DB.AutoMigrate(models...)
    log.Println("✅ Database initialized")
}

// main.go
database.InitDB(&models.User{})
```

## 🎯 Возможности

- ✅ **Модульность** - разделение на несколько YAML файлов
- ✅ **Импорт/Экспорт** - переиспользование конфигураций
- ✅ **HTTP Handlers** - декларативное описание роутов
- ✅ **Модели** - автоматическая генерация структур
- ✅ **Функции** - генерация сервисов с кодом
- ✅ **База данных** - поддержка GORM с auto-migrate
- ✅ **Пользовательский код** - поле `code` для вставки своего кода
- ✅ **Типы DNS** - `env` (os.Getenv) или `azure.env` (env.MustGet)

## 📝 TODO

- [ ] Поддержка других ORM (Aurum, GORM)
- [ ] Валидация YAML схемы
- [ ] Watch режим для авто-регенерации
- [ ] Шаблоны для разных типов проектов
- [ ] Поддержка middleware в handlers
- [ ] Генерация тестов

## 🚀 Лицензия

MIT
