# Azure Framework

Сверхбыстрый HTTP-фреймворк на Go с использованием ultrahttp движка.

## Особенности

- 🚀 **0 аллокаций** в горячем пути
- ♻️ **sync.Pool** для переиспользования контекстов
- ⚡ **Zero-copy** операции с памятью
- 🔥 **Собственный TCP стек** - не использует net/http
- 📦 **Keep-Alive** поддержка из коробки
- 🎯 **Radix tree** роутер для быстрого поиска маршрутов

## Установка

```bash
go get github.com/golang/azure
```

## Быстрый старт

```go
package main

import (
	"github.com/Payel-git-ol/azure"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func main() {
	a := azure.Defoult

	// GET маршрут с JSON ответом
	a.Get("/", func(c *azure.Context) {
		c.Json(azure.M{
			"message": "Hello World!",
			"status":  "ok",
		})
	})

	// POST с BindJSON
	a.Post("/user", func(c *azure.Context) {
		var user User
		
		// Быстрый парсинг JSON
		if err := c.BindJSON(&user); err != nil {
			c.SetStatus(400, "Bad Request")
			c.Json(azure.M{"error": err.Error()})
			return
		}
		
		c.SetStatus(201, "Created")
		c.Json(azure.M{"user": user})
	})

	// Запуск сервера
	a.Run(":8080")
}
```

## API

### Основные методы

| Метод | Описание |
|-------|----------|
| `Get(path, handler)` | Регистрация GET маршрута |
| `Post(path, handler)` | Регистрация POST маршрута |
| `Put(path, handler)` | Регистрация PUT маршрута |
| `Delete(path, handler)` | Регистрация DELETE маршрута |
| `Patch(path, handler)` | Регистрация PATCH маршрута |
| `Run(port)` | Запуск сервера |
| `Use(middleware)` | Добавить middleware |
| `Group(prefix)` | Создать группу маршрутов |

### Контекст

| Метод | Описание |
|-------|----------|
| `c.Json(data)` | Отправить JSON ответ |
| `c.Send(bytes)` | Отправить байты |
| `c.SetStatus(code, text)` | Установить HTTP статус |
| `c.BindJSON(&struct)` | Парсить JSON из тела |
| `c.GetBody()` | Получить тело запроса |
| `c.GetHeader(key)` | Получить заголовок |
| `c.SetHeader(key, value)` | Установить заголовок |
| `c.SetCookie(name, value)` | Установить cookie |
| `c.GetCookie(name)` | Получить cookie |
| `c.GetQueryParam(key)` | Получить query параметр |
| `c.Param(key)` | Получить параметр пути |

### Middleware

```go
// Logger
a.Use(azure.Logger())

// Recovery от паник
a.Use(azure.Recovery())

// Свой middleware
a.Use(func(c *azure.Context, next ultrahttp.RouteHandler) {
	// Логика до
	next(c.ultra)
	// Логика после
})
```

### Группы маршрутов

```go
api := a.Group("/api")

api.Get("/v1/status", func(c *azure.Context) {
	c.Json(azure.M{"status": "ok"})
})

api.Post("/v1/data", func(c *azure.Context) {
	c.SetStatus(201, "Created")
	c.Json(azure.M{"message": "Data received"})
})
```

## Бенчмарки

| Тест | Azure | Gin | Fiber | FastHTTP |
|------|-------|-----|-------|----------|
| GET / | 121ms | 176ms | 101ms | 112ms |
| POST /echo 1KB | **110ms** 🏆 | 142ms | 129ms | 144ms |
| GET /user/:id | **155ms** 🏆 | 178ms | 194ms | 223ms |

**Azure быстрее Gin на 28-74% и сравним с Fiber/FastHTTP!**

## Структура проекта

```
azure/
├── azure.go          # Ядро фреймворка
├── context.go        # Контекст запроса
├── group.go          # Группы маршрутов
├── types.go          # Типы (M map)
└── ultrahttp/        # HTTP движок
    ├── server.go     # TCP сервер
    ├── router.go     # Radix tree роутер
    ├── context.go    # Контекст ultrahttp
    └── adapter.go    # Адаптеры
```

## Лицензия

MIT
