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
	paramChild  *routeNode // Дочерний узел для параметров (:id)
	paramName   string     // Имя параметра для этого узла
	isParam     bool       // Флаг параметрического узла
}

// UltraRouter быстрый роутер для ultrahttp
type UltraRouter struct {
	routers [7]*routeNode // Один роутер на каждый метод

	// Not found handler
	notFound  RouteHandler
	methodMap sync.Map // Кэш для быстрого определения типа метода
}

// RouteParams параметры пути
type RouteParams struct {
	Keys   []string
	Values []string
}

// NewRouteParams создаёт новый экземпляр RouteParams
func NewRouteParams() *RouteParams {
	return &RouteParams{
		Keys:   make([]string, 0, 4),
		Values: make([]string, 0, 4),
	}
}

// GetParam получает параметр по имени
func (rp *RouteParams) GetParam(name string) string {
	for i, key := range rp.Keys {
		if key == name {
			if i < len(rp.Values) {
				return rp.Values[i]
			}
		}
	}
	return ""
}

// Reset сбрасывает параметры для переиспользования
func (rp *RouteParams) Reset() {
	rp.Keys = rp.Keys[:0]
	rp.Values = rp.Values[:0]
}

// newRouteNode создаёт новый узел
func newRouteNode(path string) *routeNode {
	return &routeNode{
		path:        path,
		children:    make([]*routeNode, 0, 4),
		childrenMap: make(map[string]*routeNode),
		priority:    0,
		paramChild:  nil,
		paramName:   "",
		isParam:     false,
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

// insert вставляет маршрут в дерево с поддержкой параметров
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

		// Проверяем, является ли сегмент параметром (:id)
		if len(segment) > 0 && segment[0] == ':' {
			paramName := segment[1:] // убираем ':'

			if current.paramChild == nil {
				current.paramChild = newRouteNode("*")
				current.paramChild.isParam = true
				current.paramChild.paramName = paramName
			}
			current = current.paramChild
			current.paramName = paramName
		} else {
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

// find ищет обработчик в дереве с поддержкой параметров
//
//go:noinline
func find(root *routeNode, path []byte, params *RouteParams) RouteHandler {
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

		segment := b2s(path[start:end])

		// Сначала ищем точное совпадение
		child, found := current.childrenMap[segment]
		if found {
			current = child
		} else if current.paramChild != nil {
			// Если точного совпадения нет, используем параметрический узел
			current = current.paramChild
			// Сохраняем параметр
			if current.paramName != "" {
				params.Keys = append(params.Keys, current.paramName)
				params.Values = append(params.Values, segment)
			}
		} else {
			return nil
		}

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
	println("DEBUG Router: method =", string(c.Request.Method), "path =", string(c.Request.Path))

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

	// Инициализируем параметры
	var params RouteParams
	params.Keys = make([]string, 0, 4)
	params.Values = make([]string, 0, 4)
	c.Request.params = &params

	// Быстрый path для root route (/)
	if len(c.Request.Path) == 1 && c.Request.Path[0] == '/' {
		println("DEBUG Router: matching root")
		if root.handler != nil {
			println("DEBUG Router: root handler found")
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
	handler := find(root, c.Request.Path, &params)

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
