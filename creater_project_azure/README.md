# Azure Project Creator

Генератор проектов на основе YAML конфигурации.

## 🚀 Быстрый старт

```bash
# Создаём проект
cd azure/creater_project_azure
go run cmd/azure-creater/main.go ../../create-api-azure.yaml
```

## 📋 Пример использования

### 1. Создаём YAML конфигурацию

```yaml
main-file: "cmd/app/main.go"

services:
  main:
    azure.use:
      - logger()
      - recovery()

    azure.handlers:
      - get("/", json("hello" -> "world"))
      - post("/users", bind(User) -> operations -> json("id" -> "created"))

  user:
    models:
      - "pkg/models/user.go" -> User

    functions:
      UserSave:
        parameters:
          name: string
          age: int
        model: User

run: "7070"
```

### 2. Генерируем проект

```bash
go run cmd/azure-creater/main.go path/to/config.yaml
```

### 3. Получаем структуру

```
your-project/
├── cmd/app/
│   └── main.go           # Сгенерированный main
├── internal/core/services/
│   ├── user.go           # Сервисы
│   └── main.go           # Функции из YAML
├── pkg/models/
│   └── user.go           # Модели
├── handlers/             # Пустая папка для хендлеров
├── middleware/           # Пустая папка для middleware
└── go.mod               # Зависимости
```

## 📖 Синтаксис YAML

### Оператор `->`

| Контекст | Значение |
|----------|----------|
| `file.go` -> `Name` | Импорт файла с именем `Name` |
| `bind(X)` -> `ops` -> `json(Y)` | Последовательность действий |
| `key` -> `value` | Пар ключ-значение в JSON |

### `azure.handlers`

```yaml
azure.handlers:
  # Простой GET
  - get("/", json("hello" -> "world"))
  
  # POST с bind и operations
  - post("/users", 
      bind(User) -> 
      operations -> 
      json("result" -> "created"))
  
  # GET с параметром
  - get("/users/:id", 
      param("id") -> 
      operations -> 
      json("user" -> "result"))
```

### `operations`

`operations` - это место для вашей кастомной логики. В сгенерированном коде это будет:

```go
// operations
// TODO: Implement your logic here
```

### `functions`

```yaml
functions:
  UserSave:
    parameters:
      name: string
      age: int
      email: string
    model: User  # Возвращает *models.User
    
  UserUpdate:
    parameters:
      id: uuid
      req: ReqUser
    returns:
      model: User  # Явное указание возврата
```

## 🔧 Генерируемый код

### Из `get("/", json("hello" -> "world"))`:

```go
a.Get("/", func(c *azure.Context) {
    c.Json(azure.M{"hello": "world"})
})
```

### Из `post("/users", bind(User) -> operations -> json("id" -> "created"))`:

```go
a.Post("/users", func(c *azure.Context) {
    // bind(User)
    var user User
    if err := c.BindJSON(&user); err != nil {
        c.JsonStatus(400, azure.M{"error": err.Error()})
        return
    }
    
    // operations
    // TODO: Implement your logic here
    
    // json("id" -> "created")
    c.Json(azure.M{"id": "created"})
})
```

### Из `functions.UserSave`:

```go
func UserSave(name string, age int) (*models.User, error) {
    // TODO: Implement logic
    return nil, nil
}
```

## 🎯 Возможности

- ✅ Автоматическое создание структуры проекта
- ✅ Генерация main.go с middleware и handlers
- ✅ Создание сервисов и функций
- ✅ Placeholder'ы для кастомной логики (`operations`)
- ✅ Модели с JSON тегами
- ✅ go.mod с зависимостями

## 📝 TODO

- [ ] Парсинг сложных выражений в handlers
- [ ] Поддержка баз данных (auto-migrate)
- [ ] Генерация миграций
- [ ] Поддержка Redis/Kafka fetcher'ов
- [ ] Валидация YAML схемы
- [ ] Watch режим для авто-регенерации

## 🚀 Лицензия

MIT
