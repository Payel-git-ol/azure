package azure

import "github.com/Payel-git-ol/azure/ultrahttp"

// Group группа маршрутов
type Group struct {
	prefix     string
	azure      *Azure
	middleware []Middleware
}

// Use добавляет middleware для группы
func (g *Group) Use(mw Middleware) {
	g.middleware = append(g.middleware, mw)
}

// Get регистрирует GET маршрут для группы
func (g *Group) Get(path string, handler func(c *Context)) {
	fullPath := g.prefix + path
	g.azure.router.GET(fullPath, g.wrapHandler(handler))
}

// Post регистрирует POST маршрут для группы
func (g *Group) Post(path string, handler func(c *Context)) {
	fullPath := g.prefix + path
	g.azure.router.POST(fullPath, g.wrapHandler(handler))
}

// Put регистрирует PUT маршрут для группы
func (g *Group) Put(path string, handler func(c *Context)) {
	fullPath := g.prefix + path
	g.azure.router.PUT(fullPath, g.wrapHandler(handler))
}

// Delete регистрирует DELETE маршрут для группы
func (g *Group) Delete(path string, handler func(c *Context)) {
	fullPath := g.prefix + path
	g.azure.router.DELETE(fullPath, g.wrapHandler(handler))
}

// Patch регистрирует PATCH маршрут для группы
func (g *Group) Patch(path string, handler func(c *Context)) {
	fullPath := g.prefix + path
	g.azure.router.PATCH(fullPath, g.wrapHandler(handler))
}

// wrapHandler для группы
func (g *Group) wrapHandler(handler func(c *Context)) ultrahttp.RouteHandler {
	return func(c *ultrahttp.Context) {
		ctx := &Context{ultra: c}

		chain := func() { handler(ctx) }
		for i := len(g.middleware) - 1; i >= 0; i-- {
			mw := g.middleware[i]
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
