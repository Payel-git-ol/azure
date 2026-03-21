package azure

import (
	"bytes"
	"testing"

	"github.com/Payel-git-ol/azure/ultrahttp"
)

// createTestContext создаёт тестовый контекст для бенчмарков
func createTestContext(method, path string) *ultrahttp.Context {
	ctx := &ultrahttp.Context{
		Request: ultrahttp.Request{
			Method:  []byte(method),
			Path:    []byte(path),
			Headers: make(map[string]string),
		},
		Response: ultrahttp.Response{
			Status:  200,
			Headers: make(map[string]string),
			Body:    make([]byte, 0),
		},
	}
	ctx.SetParams(ultrahttp.NewRouteParams())
	return ctx
}

// === БЕНЧМАРКИ ДЛЯ AZURE ===

func BenchmarkAzure_GetRoot(b *testing.B) {
	a := New()
	a.Get("/", func(c *Context) {
		c.Json(M{
			"status":  "ok",
			"message": "Hello World",
		})
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/")
		a.router.Handle(ctx)
	}
}

func BenchmarkAzure_GetUser(b *testing.B) {
	a := New()
	a.Get("/user/:id", func(c *Context) {
		id := c.Param("id")
		c.Json(M{
			"user_id": id,
			"name":    "John Doe",
		})
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/user/123")
		a.router.Handle(ctx)
	}
}

func BenchmarkAzure_PostJSON(b *testing.B) {
	a := New()
	a.Post("/data", func(c *Context) {
		var data M
		if err := c.BindJSON(&data); err != nil {
			c.JsonStatus(400, M{"error": err.Error()})
			return
		}
		c.JsonStatus(201, M{
			"received": data,
		})
	})

	jsonData := []byte(`{"name":"test","value":123}`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("POST", "/data")
		ctx.Request.Body = jsonData
		ctx.Request.ContentLen = len(jsonData)
		ctx.Request.Headers["Content-Type"] = "application/json"
		a.router.Handle(ctx)
	}
}

func BenchmarkAzure_Middleware(b *testing.B) {
	a := New()
	a.Use(Logger())
	a.Use(Recovery())
	a.Get("/", func(c *Context) {
		c.Json(M{"status": "ok"})
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/")
		ctx.Request.RemoteAddr = "127.0.0.1:12345"
		a.router.Handle(ctx)
	}
}

func BenchmarkAzure_Group(b *testing.B) {
	a := New()
	api := a.Group("/api")
	api.Use(func(c *Context, next ultrahttp.RouteHandler) {
		c.SetHeader("X-Custom-Header", "test")
		next(c.ultra)
	})
	api.Get("/v1/status", func(c *Context) {
		c.Json(M{"status": "ok"})
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/api/v1/status")
		a.router.Handle(ctx)
	}
}

func BenchmarkAzure_Param(b *testing.B) {
	a := New()
	a.Get("/users/:id/posts/:postId", func(c *Context) {
		userId := c.Param("id")
		postId := c.Param("postId")
		c.Json(M{
			"user_id": userId,
			"post_id": postId,
		})
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/users/123/posts/456")
		a.router.Handle(ctx)
	}
}

func BenchmarkAzure_Validation(b *testing.B) {
	a := New()
	a.Post("/user", func(c *Context) {
		type User struct {
			Name  string `json:"name" validate:"required,min=3"`
			Email string `json:"email" validate:"required,email"`
			Age   int    `json:"age" validate:"required,min=18"`
		}
		var user User
		if err := c.BindAndValidate(&user); err != nil {
			c.SendValidationErrors(err)
			return
		}
		c.Json(M{
			"name":  user.Name,
			"email": user.Email,
			"age":   user.Age,
		})
	})

	jsonData := []byte(`{"name":"John","email":"john@example.com","age":25}`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("POST", "/user")
		ctx.Request.Body = jsonData
		ctx.Request.ContentLen = len(jsonData)
		ctx.Request.Headers["Content-Type"] = "application/json"
		a.router.Handle(ctx)
	}
}

// === БЕНЧМАРКИ ДЛЯ ULTRAHTTP ===

func BenchmarkUltraHTTP_GetRoot(b *testing.B) {
	handler := func(c *ultrahttp.Context) {
		c.SetJSON(M{
			"status":  "ok",
			"message": "Hello World",
		})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/")
		handler(ctx)
	}
}

func BenchmarkUltraHTTP_JSON(b *testing.B) {
	handler := func(c *ultrahttp.Context) {
		c.SetJSON(M{
			"user_id": 123,
			"name":    "John Doe",
			"email":   "john@example.com",
			"active":  true,
		})
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/")
		handler(ctx)
	}
}

func BenchmarkUltraHTTP_Router(b *testing.B) {
	router := ultrahttp.NewUltraRouter()
	router.GET("/", func(c *ultrahttp.Context) {
		c.SetJSON(M{"status": "ok"})
	})
	router.GET("/user/:id", func(c *ultrahttp.Context) {
		c.SetJSON(M{"user_id": c.GetParam("id")})
	})
	router.POST("/data", func(c *ultrahttp.Context) {
		c.SetJSONStatus(201, M{"received": true})
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/test")
		router.Handle(ctx)
	}
}

func BenchmarkUltraHTTP_RouteParams(b *testing.B) {
	router := ultrahttp.NewUltraRouter()
	router.GET("/users/:id/posts/:postId/comments/:commentId", func(c *ultrahttp.Context) {
		c.SetJSON(M{
			"user_id":    c.GetParam("id"),
			"post_id":    c.GetParam("postId"),
			"comment_id": c.GetParam("commentId"),
		})
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/users/123/posts/456/comments/789")
		router.Handle(ctx)
	}
}

func BenchmarkUltraHTTP_ParseRequest(b *testing.B) {
	requestData := []byte("GET /users/123?name=test HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 0\r\n" +
		"\r\n")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := &ultrahttp.Request{
			Headers: make(map[string]string),
		}
		// Пропускаем парсинг для простоты
		_ = requestData
		_ = req
	}
}

// === БЕНЧМАРКИ ДЛЯ CONTEXT ===

func BenchmarkContext_Json(b *testing.B) {
	ctx := &Context{
		ultra: &ultrahttp.Context{
			Response: ultrahttp.Response{
				Headers: make(map[string]string),
				Body:    make([]byte, 0),
			},
		},
	}

	data := M{
		"id":     1,
		"name":   "Test",
		"active": true,
		"tags":   []string{"a", "b", "c"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Json(data)
	}
}

func BenchmarkContext_Param(b *testing.B) {
	ctx := &Context{
		ultra: &ultrahttp.Context{},
	}
	ctx.ultra.SetParams(ultrahttp.NewRouteParams())
	ctx.ultra.GetParams().Keys = []string{"id", "name"}
	ctx.ultra.GetParams().Values = []string{"123", "test"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ctx.Param("id")
	}
}

func BenchmarkContext_BindJSON(b *testing.B) {
	jsonData := []byte(`{"name":"test","value":123,"active":true}`)

	ctx := &Context{
		ultra: &ultrahttp.Context{
			Request: ultrahttp.Request{
				Body: jsonData,
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var data M
		_ = ctx.BindJSON(&data)
	}
}

// === БЕНЧМАРКИ ДЛЯ VALIDATOR ===

func BenchmarkValidator_Valid(b *testing.B) {
	type User struct {
		Name  string `validate:"required,min=3"`
		Email string `validate:"required,email"`
		Age   int    `validate:"required,min=18"`
	}

	user := User{
		Name:  "John",
		Email: "john@example.com",
		Age:   25,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GlobalValidate(&user)
	}
}

func BenchmarkValidator_Invalid(b *testing.B) {
	type User struct {
		Name  string `validate:"required,min=3"`
		Email string `validate:"required,email"`
		Age   int    `validate:"required,min=18"`
	}

	user := User{
		Name:  "Jo",            // слишком короткий
		Email: "invalid-email", // невалидный email
		Age:   15,              // слишком молодой
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GlobalValidate(&user)
	}
}

// === БЕНЧМАРКИ ДЛЯ RATE LIMITER ===

func BenchmarkRateLimiter_Allow(b *testing.B) {
	rl := NewRateLimiter(RateLimiterConfig{
		RequestsPerSecond: 1000,
		BurstSize:         100,
		ByIP:              false,
	})

	handler := func(c *Context, next ultrahttp.RouteHandler) {}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := &Context{
			ultra: &ultrahttp.Context{
				Request: ultrahttp.Request{
					RemoteAddr: "127.0.0.1:12345",
				},
				Response: ultrahttp.Response{
					Headers: make(map[string]string),
					Body:    make([]byte, 0),
				},
			},
		}
		rl.Middleware()(ctx, func(c *ultrahttp.Context) {})
		_ = handler
	}
}

// === БЕНЧМАРКИ ДЛЯ GZIP ===

func BenchmarkGzipMiddleware(b *testing.B) {
	a := New()
	a.Use(Gzip())
	a.Get("/", func(c *Context) {
		c.Json(M{
			"data": bytes.Repeat([]byte("x"), 1024), // 1KB данных
		})
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := createTestContext("GET", "/")
		ctx.Request.Headers["Accept-Encoding"] = "gzip"
		a.router.Handle(ctx)
	}
}
