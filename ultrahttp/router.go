package ultrahttp

import (
	"sync"
)

// === БЫСТРЫЙ РОУТЕР ДЛЯ ULTRAHTTP (Radix Tree) ===

// HTTP методы как константы
const (
	methodTypeGET     = 0
	methodTypePOST    = 1
	methodTypePUT     = 2
	methodTypeDELETE  = 3
	methodTypePATCH   = 4
	methodTypeHEAD    = 5
	methodTypeOPTIONS = 6
	methodTypeUNKNOWN = 7
)

// RouteHandler обработчик для маршрута
type RouteHandler func(c *Context)

// routeNode узел radix дерева
type routeNode struct {
	path        string
	handler     RouteHandler
	children    []*routeNode
	childrenMap map[string]*routeNode // Быстрый поиск за O(1)
	priority    uint32
}

// UltraRouter быстрый роутер для ultrahttp
type UltraRouter struct {
	routers [7]*routeNode // Один роутер на каждый метод

	// Not found handler
	notFound  RouteHandler
	methodMap sync.Map // Кэш для быстрого определения типа метода
}

// newRouteNode создаёт новый узел
func newRouteNode(path string) *routeNode {
	return &routeNode{
		path:        path,
		children:    make([]*routeNode, 0, 4),
		childrenMap: make(map[string]*routeNode),
		priority:    0,
	}
}

// NewUltraRouter создаёт новый роутер
func NewUltraRouter() *UltraRouter {
	return &UltraRouter{
		routers: [7]*routeNode{
			newRouteNode("/"), // GET
			newRouteNode("/"), // POST
			newRouteNode("/"), // PUT
			newRouteNode("/"), // DELETE
			newRouteNode("/"), // PATCH
			newRouteNode("/"), // HEAD
			newRouteNode("/"), // OPTIONS
		},
	}
}

// insert вставляет маршрут в дерево
func insert(root *routeNode, path string, handler RouteHandler) {
	if path == "/" {
		root.handler = handler
		return
	}

	current := root
	segments := splitPath(path)

	for _, segment := range segments {
		if segment == "" {
			continue
		}

		// Ищем в map для O(1) поиска
		child, found := current.childrenMap[segment]
		if found {
			child.priority++
			current = child
		} else {
			// Создаём новый узел если не найден
			newNode := newRouteNode(segment)
			newNode.priority = 1
			current.childrenMap[segment] = newNode
			current = newNode
		}
	}

	current.handler = handler
}

// splitPath быстро разбивает путь на сегменты
func splitPath(path string) []string {
	if path == "" {
		return nil
	}

	// Считаем количество сегментов
	count := 1
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			count++
		}
	}

	result := make([]string, 0, count)
	start := 0
	for i := 0; i <= len(path); i++ {
		if i == len(path) || path[i] == '/' {
			if i > start {
				result = append(result, path[start:i])
			}
			start = i + 1
		}
	}

	return result
}

// find ищет обработчик в дереве
//
//go:noinline
func find(root *routeNode, path []byte) RouteHandler {
	if len(path) == 0 || path[0] != '/' {
		return nil
	}

	if len(path) == 1 {
		return root.handler
	}

	current := root
	start := 1

	for start < len(path) {
		end := start
		for end < len(path) && path[end] != '/' {
			end++
		}

		// Конвертируем segment в string для lookup в map (одноразово)
		// но это быстрее чем линейный поиск через массив
		segment := b2s(path[start:end])

		child, found := current.childrenMap[segment]
		if !found {
			return nil
		}

		current = child
		start = end + 1
	}

	return current.handler
}

// GET регистрирует GET маршрут
func (r *UltraRouter) GET(path string, handler RouteHandler) {
	insert(r.routers[methodTypeGET], path, handler)
}

// POST регистрирует POST маршрут
func (r *UltraRouter) POST(path string, handler RouteHandler) {
	insert(r.routers[methodTypePOST], path, handler)
}

// PUT регистрирует PUT маршрут
func (r *UltraRouter) PUT(path string, handler RouteHandler) {
	insert(r.routers[methodTypePUT], path, handler)
}

// DELETE регистрирует DELETE маршрут
func (r *UltraRouter) DELETE(path string, handler RouteHandler) {
	insert(r.routers[methodTypeDELETE], path, handler)
}

// PATCH регистрирует PATCH маршрут
func (r *UltraRouter) PATCH(path string, handler RouteHandler) {
	insert(r.routers[methodTypePATCH], path, handler)
}

// HEAD регистрирует HEAD маршрут
func (r *UltraRouter) HEAD(path string, handler RouteHandler) {
	insert(r.routers[methodTypeHEAD], path, handler)
}

// OPTIONS регистрирует OPTIONS маршрут
func (r *UltraRouter) OPTIONS(path string, handler RouteHandler) {
	insert(r.routers[methodTypeOPTIONS], path, handler)
}

// getMethodType определяет тип метода быстро
//
//go:noinline
func getMethodType(method []byte) int {
	switch {
	case bytesEqualConst(method, "GET"):
		return methodTypeGET
	case bytesEqualConst(method, "POST"):
		return methodTypePOST
	case bytesEqualConst(method, "PUT"):
		return methodTypePUT
	case bytesEqualConst(method, "DELETE"):
		return methodTypeDELETE
	case bytesEqualConst(method, "PATCH"):
		return methodTypePATCH
	case bytesEqualConst(method, "HEAD"):
		return methodTypeHEAD
	case bytesEqualConst(method, "OPTIONS"):
		return methodTypeOPTIONS
	default:
		return methodTypeUNKNOWN
	}
}

// bytesEqualConst быстрое сравнение с константной строкой
//
//go:noinline
func bytesEqualConst(b []byte, s string) bool {
	if len(b) != len(s) {
		return false
	}
	for i := 0; i < len(b); i++ {
		if b[i] != s[i] {
			return false
		}
	}
	return true
}

// Handle обрабатывает запрос
func (r *UltraRouter) Handle(c *Context) {
	// Быстрое определение типа метода
	methodType := getMethodType(c.Request.Method)

	if methodType == methodTypeUNKNOWN {
		if r.notFound != nil {
			r.notFound(c)
		} else {
			c.SendNotFound()
		}
		return
	}

	// Получаем роутер для данного метода
	root := r.routers[methodType]

	// Быстрый path для root route (/)
	if len(c.Request.Path) == 1 && c.Request.Path[0] == '/' {
		if root.handler != nil {
			root.handler(c)
			return
		}

		if r.notFound != nil {
			r.notFound(c)
		} else {
			c.SendNotFound()
		}
		return
	}

	// Ищем handler в дереве для других путей
	handler := find(root, c.Request.Path)

	if handler != nil {
		handler(c)
		return
	}

	// Not found
	if r.notFound != nil {
		r.notFound(c)
	} else {
		c.SendNotFound()
	}
}

// NotFound устанавливает обработчик для 404
func (r *UltraRouter) NotFound(handler RouteHandler) {
	r.notFound = handler
}

// Any регистрирует маршрут для всех методов
func (r *UltraRouter) Any(path string, handler RouteHandler) {
	r.GET(path, handler)
	r.POST(path, handler)
	r.PUT(path, handler)
	r.DELETE(path, handler)
	r.PATCH(path, handler)
}

// Group создаёт группу маршрутов с префиксом
func (r *UltraRouter) Group(prefix string) *UltraGroup {
	return &UltraGroup{
		prefix: prefix,
		router: r,
	}
}

// UltraGroup группа маршрутов
type UltraGroup struct {
	prefix string
	router *UltraRouter
}

// GET регистрирует GET маршрут для группы
func (g *UltraGroup) GET(path string, handler RouteHandler) {
	g.router.GET(g.prefix+path, handler)
}

// POST регистрирует POST маршрут для группы
func (g *UltraGroup) POST(path string, handler RouteHandler) {
	g.router.POST(g.prefix+path, handler)
}

// PUT регистрирует PUT маршрут для группы
func (g *UltraGroup) PUT(path string, handler RouteHandler) {
	g.router.PUT(g.prefix+path, handler)
}

// DELETE регистрирует DELETE маршрут для группы
func (g *UltraGroup) DELETE(path string, handler RouteHandler) {
	g.router.DELETE(g.prefix+path, handler)
}

// PATCH регистрирует PATCH маршрут для группы
func (g *UltraGroup) PATCH(path string, handler RouteHandler) {
	g.router.PATCH(g.prefix+path, handler)
}
