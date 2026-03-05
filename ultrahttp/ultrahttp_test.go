package ultrahttp

import (
	"testing"
)

func BenchmarkUltraRouter_Get(b *testing.B) {
	router := NewUltraRouter()
	router.GET("/test", func(c *Context) {
		c.SetText("ok")
	})

	ctx := &Context{
		Request: Request{
			Method:  methodGET,
			Path:    []byte("/test"),
			Headers: make(map[string]string, 16),
		},
		Response: Response{
			Headers: make(map[string]string, 8),
			buffer:  make([]byte, DefaultBufferSize),
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		router.Handle(ctx)
		ctx.Response.Body = ctx.Response.Body[:0]
	}
}

func BenchmarkUltraRouter_Post(b *testing.B) {
	router := NewUltraRouter()
	router.POST("/api/user", func(c *Context) {
		c.SetStatus(201, "Created")
		c.SetText("created")
	})

	ctx := &Context{
		Request: Request{
			Method:  methodPOST,
			Path:    []byte("/api/user"),
			Headers: make(map[string]string, 16),
		},
		Response: Response{
			Headers: make(map[string]string, 8),
			buffer:  make([]byte, DefaultBufferSize),
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		router.Handle(ctx)
		ctx.Response.Body = ctx.Response.Body[:0]
	}
}

func BenchmarkUltraRouter_MultipleRoutes(b *testing.B) {
	router := NewUltraRouter()

	routes := []string{
		"/",
		"/users",
		"/users/profile",
		"/users/settings",
		"/api/v1/data",
		"/api/v1/items",
		"/api/v2/data",
		"/static/css/main.css",
		"/static/js/app.js",
	}

	for _, route := range routes {
		router.GET(route, func(c *Context) {
			c.SetText("ok")
		})
	}

	ctx := &Context{
		Request: Request{
			Method:  methodGET,
			Path:    []byte("/test"),
			Headers: make(map[string]string, 16),
		},
		Response: Response{
			Headers: make(map[string]string, 8),
			buffer:  make([]byte, DefaultBufferSize),
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, route := range routes {
			ctx.Request.Path = []byte(route)
			router.Handle(ctx)
			ctx.Response.Body = ctx.Response.Body[:0]
		}
	}
}

func BenchmarkContext_SetJSON(b *testing.B) {
	ctx := &Context{
		Response: Response{
			Headers: make(map[string]string, 8),
			buffer:  make([]byte, DefaultBufferSize),
		},
	}

	data := M{
		"message": "hello",
		"status":  "ok",
		"count":   42,
		"name":    "test",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.SetJSON(data)
		ctx.Response.Body = ctx.Response.Body[:0]
	}
}

func BenchmarkContext_SetText(b *testing.B) {
	ctx := &Context{
		Response: Response{
			Headers: make(map[string]string, 8),
			buffer:  make([]byte, DefaultBufferSize),
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx.SetText("Hello World!")
		ctx.Response.Body = ctx.Response.Body[:0]
	}
}

func BenchmarkContext_GetQueryParam(b *testing.B) {
	ctx := &Context{
		Request: Request{
			QueryString: []byte("q=hello&limit=10&offset=20"),
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ctx.GetQueryParam("q")
		_ = ctx.GetQueryParam("limit")
		_ = ctx.GetQueryParam("offset")
	}
}

func BenchmarkContext_GetCookie(b *testing.B) {
	ctx := &Context{
		Request: Request{
			Headers: map[string]string{
				"Cookie": "session=abc123; user=john; token=xyz789",
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ctx.GetCookie("session")
		_, _ = ctx.GetCookie("user")
		_, _ = ctx.GetCookie("token")
	}
}

func BenchmarkBytesEqual(b *testing.B) {
	a := []byte("/api/v1/users/profile")
	b2 := []byte("/api/v1/users/profile")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = bytesEqual(a, b2)
	}
}

func BenchmarkAppendInt(b *testing.B) {
	buf := make([]byte, 0, 32)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf = appendInt(buf[:0], 12345)
	}
}

func TestUltraRouter_Get(t *testing.T) {
	router := NewUltraRouter()
	called := false

	router.GET("/test", func(c *Context) {
		called = true
		c.SetText("ok")
	})

	ctx := &Context{
		Request: Request{
			Method:  methodGET,
			Path:    []byte("/test"),
			Headers: make(map[string]string, 16),
		},
		Response: Response{
			Status:     200,
			StatusText: "OK",
			Headers:    make(map[string]string, 8),
			buffer:     make([]byte, DefaultBufferSize),
		},
	}

	router.Handle(ctx)

	if !called {
		t.Error("Handler was not called")
	}

	if ctx.Response.Status != 200 {
		t.Errorf("Expected status 200, got %d", ctx.Response.Status)
	}
}

func TestUltraRouter_NotFound(t *testing.T) {
	router := NewUltraRouter()

	ctx := &Context{
		Request: Request{
			Method:  methodGET,
			Path:    []byte("/notfound"),
			Headers: make(map[string]string, 16),
		},
		Response: Response{
			Headers: make(map[string]string, 8),
			buffer:  make([]byte, DefaultBufferSize),
		},
	}

	router.Handle(ctx)

	if ctx.Response.Status != 404 {
		t.Errorf("Expected status 404, got %d", ctx.Response.Status)
	}
}

func TestContext_SetJSON(t *testing.T) {
	ctx := &Context{
		Response: Response{
			Headers: make(map[string]string, 8),
		},
	}

	ctx.SetJSON(M{
		"hello": "world",
	})

	if ctx.Response.Headers["Content-Type"] != "application/json; charset=utf-8" {
		t.Errorf("Expected JSON content type, got %s", ctx.Response.Headers["Content-Type"])
	}

	if len(ctx.Response.Body) == 0 {
		t.Error("Expected non-empty body")
	}
}

func TestContext_GetQueryParam(t *testing.T) {
	ctx := &Context{
		Request: Request{
			QueryString: []byte("q=hello&limit=10&offset=20"),
		},
	}

	q := ctx.GetQueryParam("q")
	if q != "hello" {
		t.Errorf("Expected query 'hello', got '%s'", q)
	}

	limit := ctx.GetQueryParam("limit")
	if limit != "10" {
		t.Errorf("Expected limit '10', got '%s'", limit)
	}
}

func TestContext_GetCookie(t *testing.T) {
	ctx := &Context{
		Request: Request{
			Headers: map[string]string{
				"Cookie": "session=abc123; user=john",
			},
		},
	}

	session, found := ctx.GetCookie("session")
	if !found {
		t.Error("Expected cookie 'session' to be found")
	}
	if session != "abc123" {
		t.Errorf("Expected session 'abc123', got '%s'", session)
	}

	user, found := ctx.GetCookie("user")
	if !found {
		t.Error("Expected cookie 'user' to be found")
	}
	if user != "john" {
		t.Errorf("Expected user 'john', got '%s'", user)
	}
}

func TestContext_Group(t *testing.T) {
	router := NewUltraRouter()
	called := false

	api := router.Group("/api")
	api.GET("/v1/status", func(c *Context) {
		called = true
		c.SetText("ok")
	})

	ctx := &Context{
		Request: Request{
			Method:  methodGET,
			Path:    []byte("/api/v1/status"),
			Headers: make(map[string]string, 16),
		},
		Response: Response{
			Status:     200,
			StatusText: "OK",
			Headers:    make(map[string]string, 8),
			buffer:     make([]byte, DefaultBufferSize),
		},
	}

	router.Handle(ctx)

	if !called {
		t.Error("Group handler was not called")
	}
}
