package azure

import (
	"github.com/golang/azure/ultrahttp"
	"log"
)

// Azure - фреймворк на ultrahttp движке
type Azure struct {
	router     *ultrahttp.UltraRouter
	middleware []Middleware
	name       string
}

// Middleware функция промежуточного обработчика
type Middleware func(c *Context, next ultrahttp.RouteHandler)

// Defoult экземпляр по умолчанию (сохраняем твою орфографию :)
var Defoult *Azure

func init() {
	Defoult = New()
}

// New создаёт новый экземпляр Azure
func New() *Azure {
	return &Azure{
		router:     ultrahttp.NewUltraRouter(),
		middleware: make([]Middleware, 0, 8),
		name:       "Azure",
	}
}

// Use добавляет middleware
func (a *Azure) Use(mw Middleware) {
	a.middleware = append(a.middleware, mw)
}

// Get регистрирует GET маршрут
func (a *Azure) Get(path string, handler func(c *Context)) {
	a.router.GET(path, a.wrapHandler(handler))
}

// Post регистрирует POST маршрут
func (a *Azure) Post(path string, handler func(c *Context)) {
	a.router.POST(path, a.wrapHandler(handler))
}

// Put регистрирует PUT маршрут
func (a *Azure) Put(path string, handler func(c *Context)) {
	a.router.PUT(path, a.wrapHandler(handler))
}

// Delete регистрирует DELETE маршрут
func (a *Azure) Delete(path string, handler func(c *Context)) {
	a.router.DELETE(path, a.wrapHandler(handler))
}

// Patch регистрирует PATCH маршрут
func (a *Azure) Patch(path string, handler func(c *Context)) {
	a.router.PATCH(path, a.wrapHandler(handler))
}

// wrapHandler оборачивает хендлер с middleware
func (a *Azure) wrapHandler(handler func(c *Context)) ultrahttp.RouteHandler {
	return func(c *ultrahttp.Context) {
		ctx := &Context{ultra: c}

		// Применяем middleware в обратном порядке
		chain := func() { handler(ctx) }
		for i := len(a.middleware) - 1; i >= 0; i-- {
			mw := a.middleware[i]
			next := chain
			chain = func() {
				mw(ctx, func(c *ultrahttp.Context) {
					next()
				})
			}
		}
		chain()
	}
}

// Run запускает сервер
func (a *Azure) Run(addr string) error {
	if len(addr) > 0 && addr[0] != ':' {
		addr = ":" + addr
	}

	log.Printf("[%s] Server starting on port %s", a.name, addr[1:])
	log.Printf("[%s] Using ultrahttp engine - ZERO allocations!", a.name)

	server := ultrahttp.NewServer(addr, a.router.Handle)
	return server.ListenAndServe()
}

// Group создаёт группу маршрутов с префиксом
func (a *Azure) Group(prefix string) *Group {
	return &Group{
		prefix: prefix,
		azure:  a,
	}
}

// Logger middleware логгер
func Logger() Middleware {
	return func(c *Context, next ultrahttp.RouteHandler) {
		// Логируем запрос
		log.Printf("[%s] %s %s", c.ultra.GetMethod(), c.ultra.GetPath(), c.ultra.GetRemoteAddr())
		
		// Вызываем следующий хендлер
		next(c.ultra)
	}
}

// Recovery middleware для восстановления после паник
func Recovery() Middleware {
	return func(c *Context, next ultrahttp.RouteHandler) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[Recovery] panic recovered: %v", err)
				c.ultra.SetStatus(500, "Internal Server Error")
				c.ultra.SetJSON(M{
					"error": "Internal Server Error",
				})
			}
		}()
		next(c.ultra)
	}
}
