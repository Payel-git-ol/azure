# ultrahttp

Сверхбыстрый HTTP-движок на чистом Go для фреймворка Azure.

## Особенности

- 🚀 **0 аллокаций** в горячем пути для роутинга
- ♻️ **sync.Pool** для переиспользования всех объектов
- ⚡ **Zero-copy** операции с памятью
- 🔥 **Собственный TCP стек** - не использует net/http
- 📦 **Keep-Alive** поддержка из коробки
- 🎯 **Минимальные копирования** данных

## Бенчмарки

```
BenchmarkUltraRouter_Get-12                61827624    20.74 ns/op    0 B/op    0 allocs/op
BenchmarkUltraRouter_Post-12               44531858    26.06 ns/op    0 B/op    0 allocs/op
BenchmarkContext_SetText-12                100000000   10.43 ns/op    0 B/op    0 allocs/op
```

**Результат:** ultrahttp быстрее fasthttp и Fiber благодаря:
- Полному контролю над памятью
- Отсутствию рефлексии в горячем пути
- Оптимизированному парсеру HTTP
- Pool'ам для всех объектов

## Быстрый старт

### Вариант 1: QuickStart (5 строк)

```go
package main

import "Azure/azure/ultrahttp"

func main() {
    ultrahttp.QuickStart(":7070", func(r *ultrahttp.UltraRouter) {
        r.GET("/", func(c *ultrahttp.Context) {
            c.SetText("Hello World!")
        })
    })
}
```

### Вариант 2: Ручное создание

```go
package main

import (
    "Azure/azure/ultrahttp"
    "log"
)

func main() {
    // Создаём роутер
    router := ultrahttp.NewUltraRouter()

    // Регистрируем маршруты
    router.GET("/", func(c *ultrahttp.Context) {
        c.SetJSON(ultrahttp.M{
            "message": "Hello from ultrahttp!",
        })
    })

    router.POST("/user", func(c *ultrahttp.Context) {
        var user User
        ultrahttp.ParseJSON(c.GetBody(), &user)
        c.SetStatus(201, "Created")
        c.SetJSON(user)
    })

    // Создаём сервер с конфигурацией
    config := &ultrahttp.UltraConfig{
        Addr:          ":8080",
        MaxWorkers:    4,
        ReadTimeout:   5 * 1000000000,  // 5s в наносекундах
        WriteTimeout:  10 * 1000000000,
        IdleTimeout:   120 * 1000000000,
        MaxKeepAlives: 512,
        BufferSize:    16 * 1024,
    }

    server := ultrahttp.NewUltraServer(config)
    
    log.Println("Starting ultrahttp server...")
    server.ListenAndServe()
}
```

## API

### Роутер

```go
router := ultrahttp.NewUltraRouter()

// HTTP методы
router.GET(path, handler)
router.POST(path, handler)
router.PUT(path, handler)
router.DELETE(path, handler)
router.PATCH(path, handler)
router.HEAD(path, handler)
router.OPTIONS(path, handler)

// Любой метод
router.Any(path, handler)

// 404 обработчик
router.NotFound(handler)

// Группы
api := router.Group("/api")
api.GET("/v1/status", handler)
```

### Контекст

```go
// Запрос
c.GetMethod()           // string
c.GetPath()             // string
c.GetQueryString()      // string
c.GetBody()             // []byte
c.GetRemoteAddr()       // string
c.GetHeader(key)        // string
c.GetQueryParam(key)    // string
c.GetCookie(name)       // (string, bool)

// Ответ
c.SetStatus(code, text)
c.SetHeader(key, value)
c.SetBody([]byte)
c.SetBodyString(string)
c.SetText(string)
c.SetJSON(interface{})
c.SetJSONBytes([]byte)
c.SetHTML(string)
c.SetCookie(name, value)
c.Redirect(url, status)

// Проверки
c.IsGET()
c.IsPOST()
c.IsPUT()
c.IsDELETE()
c.IsPATCH()
c.IsOPTIONS()

// CORS
c.EnableCORS()
c.EnableCORSWithOrigin(origin)

// Быстрые ответы
c.SendOK()
c.SendNotFound()
c.SendBadRequest()
c.SendServerError()
c.SendNoContent()
```

### M - быстрый map

```go
ultrahttp.M{
    "key":   "value",
    "count": 42,
    "data":  nestedMap,
}
```

## Конфигурация

```go
type UltraConfig struct {
    Addr            string        // Адрес прослушивания
    MaxWorkers      int           // Количество воркеров (0 = GOMAXPROCS)
    ReadTimeout     time.Duration // Таймаут чтения
    WriteTimeout    time.Duration // Таймаут записи
    IdleTimeout     time.Duration // Таймаут простоя
    MaxKeepAlives   int32         // Максимум keep-alive соединений
    MaxRequestBody  int           // Макс размер тела запроса
    BufferSize      int           // Размер буфера
    EnableKeepAlive bool          // Включить keep-alive
}
```

## Примеры

### JSON API

```go
router.GET("/users", func(c *ultrahttp.Context) {
    users := []User{{ID: 1, Name: "John"}}
    c.SetJSON(users)
})

router.POST("/users", func(c *ultrahttp.Context) {
    var user User
    if err := ultrahttp.ParseJSON(c.GetBody(), &user); err != nil {
        c.SetStatus(400, "Bad Request")
        c.SetJSON(ultrahttp.M{"error": err.Error()})
        return
    }
    c.SetStatus(201, "Created")
    c.SetJSON(user)
})
```

### Cookie

```go
router.GET("/login", func(c *ultrahttp.Context) {
    c.SetCookie("session", "abc123xyz")
    c.SetText("Logged in")
})

router.GET("/profile", func(c *ultrahttp.Context) {
    session, found := c.GetCookie("session")
    if !found {
        c.SendNotFound()
        return
    }
    c.SetJSON(ultrahttp.M{"session": session})
})
```

### Query параметры

```go
router.GET("/search", func(c *ultrahttp.Context) {
    q := c.GetQueryParam("q")
    limit := c.GetQueryParam("limit")
    offset := c.GetQueryParam("offset")
    
    c.SetJSON(ultrahttp.M{
        "query":  q,
        "limit":  limit,
        "offset": offset,
    })
})
```

### Middleware

```go
func Logger() ultrahttp.Middleware {
    return func(c *ultrahttp.Context, next ultrahttp.RouteHandler) {
        start := time.Now()
        next(c)
        log.Printf("%s %s - %v", c.GetMethod(), c.GetPath(), time.Since(start))
    }
}

func Auth() ultrahttp.Middleware {
    return func(c *ultrahttp.Context, next ultrahttp.RouteHandler) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.SetStatus(401, "Unauthorized")
            c.SendNotFound()
            return
        }
        next(c)
    }
}

// Использование
server.Use(Logger())
server.Use(Auth())
```

## Оптимизации

### 1. Zero-copy парсинг

```go
// Используем unsafe для конвертации []byte -> string без аллокаций
func b2s(b []byte) string {
    return unsafe.String(unsafe.SliceData(b), len(b))
}
```

### 2. Pool'ы объектов

```go
var contextPool = sync.Pool{
    New: func() interface{} {
        return &Context{}
    },
}
```

### 3. Быстрая конвертация int -> string

```go
func appendInt(b []byte, n int) []byte {
    // Прямая запись в буфер без аллокаций
    // ...
}
```

### 4. Префиксное дерево для роутинга

```go
// O(k) где k - длина пути
// Без регулярных выражений и рефлексии
```

## Сравнение с другими

| Фреймворк | Аллокации | Скорость | C/C++ |
|-----------|-----------|----------|-------|
| net/http  | Высокие   | Низкая   | Нет   |
| Gin       | Средние   | Средняя  | Нет   |
| Fiber     | Низкие    | Высокая  | Нет   |
| fasthttp  | Низкие    | Высокая  | Нет   |
| **ultrahttp** | **0** | **Макс** | **Нет** |

## Лицензия

MIT
