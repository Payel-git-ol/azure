package ultrahttp

import (
	"sync/atomic"
	"time"
)

// === АДАПТЕР ДЛЯ ИНТЕГРАЦИИ С AZURE ===

// AzureAdapter адаптирует azure роутер для ultrahttp
type AzureAdapter struct {
	router *UltraRouter
}

// NewAzureAdapter создаёт адаптер
func NewAzureAdapter() *AzureAdapter {
	return &AzureAdapter{
		router: NewUltraRouter(),
	}
}

// Get регистрирует GET маршрут
func (a *AzureAdapter) Get(path string, handler func(c *Context)) {
	a.router.GET(path, handler)
}

// Post регистрирует POST маршрут
func (a *AzureAdapter) Post(path string, handler func(c *Context)) {
	a.router.POST(path, handler)
}

// Put регистрирует PUT маршрут
func (a *AzureAdapter) Put(path string, handler func(c *Context)) {
	a.router.PUT(path, handler)
}

// Delete регистрирует DELETE маршрут
func (a *AzureAdapter) Delete(path string, handler func(c *Context)) {
	a.router.DELETE(path, handler)
}

// Patch регистрирует PATCH маршрут
func (a *AzureAdapter) Patch(path string, handler func(c *Context)) {
	a.router.PATCH(path, handler)
}

// Serve возвращает handler для ultrahttp.Server
func (a *AzureAdapter) Serve() Handler {
	return a.router.Handle
}

// Run запускает сервер с адаптером
func (a *AzureAdapter) Run(addr string) error {
	server := NewServer(addr, a.Serve())
	return server.ListenAndServe()
}

// === ULTRA SERVER С ДОПОЛНИТЕЛЬНЫМИ ОПЦИЯМИ ===

// UltraServer расширенный сервер с опциями
type UltraServer struct {
	*Server
	router *UltraRouter
}

// UltraConfig конфигурация сервера
type UltraConfig struct {
	Addr            string
	MaxWorkers      int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	MaxKeepAlives   int32
	MaxRequestBody  int
	BufferSize      int
	EnableKeepAlive bool
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *UltraConfig {
	return &UltraConfig{
		MaxWorkers:      0, // GOMAXPROCS
		ReadTimeout:     DefaultReadTimeout,
		WriteTimeout:    DefaultWriteTimeout,
		IdleTimeout:     DefaultIdleTimeout,
		MaxKeepAlives:   DefaultMaxKeepAlives,
		MaxRequestBody:  MaxRequestBodySize,
		BufferSize:      DefaultBufferSize,
		EnableKeepAlive: true,
	}
}

// NewUltraServer создаёт новый ultra сервер с роутером
func NewUltraServer(config *UltraConfig) *UltraServer {
	if config == nil {
		config = DefaultConfig()
	}

	router := NewUltraRouter()
	
	server := &Server{
		addr:          config.Addr,
		handler:       router.Handle,
		maxWorkers:    config.MaxWorkers,
		readTimeout:   config.ReadTimeout,
		writeTimeout:  config.WriteTimeout,
		idleTimeout:   config.IdleTimeout,
		maxKeepAlives: config.MaxKeepAlives,
	}

	return &UltraServer{
		Server: server,
		router: router,
	}
}

// GET регистрирует маршрут
func (s *UltraServer) GET(path string, handler RouteHandler) {
	s.router.GET(path, handler)
}

// POST регистрирует маршрут
func (s *UltraServer) POST(path string, handler RouteHandler) {
	s.router.POST(path, handler)
}

// PUT регистрирует маршрут
func (s *UltraServer) PUT(path string, handler RouteHandler) {
	s.router.PUT(path, handler)
}

// DELETE регистрирует маршрут
func (s *UltraServer) DELETE(path string, handler RouteHandler) {
	s.router.DELETE(path, handler)
}

// PATCH регистрирует маршрут
func (s *UltraServer) PATCH(path string, handler RouteHandler) {
	s.router.PATCH(path, handler)
}

// Group создаёт группу маршрутов
func (s *UltraServer) Group(prefix string) *UltraGroup {
	return s.router.Group(prefix)
}

// NotFound устанавливает обработчик 404
func (s *UltraServer) NotFound(handler RouteHandler) {
	s.router.NotFound(handler)
}

// Use middleware (заглушка для будущей реализации)
func (s *UltraServer) Use(middleware func(c *Context, next RouteHandler)) {
	// TODO: реализовать middleware chain
}

// === БЫСТРЫЙ ЗАПУСК ===

// QuickStart быстрый запуск сервера
func QuickStart(addr string, setup func(r *UltraRouter)) error {
	router := NewUltraRouter()
	setup(router)
	
	server := NewServer(addr, router.Handle)
	return server.ListenAndServe()
}

// === СТАТИСТИКА ===

// Stats статистика сервера
type Stats struct {
	ActiveConnections int32
	TotalRequests     int64
	TotalErrors       int64
}

var (
	totalRequests int64
	totalErrors   int64
)

// GetStats возвращает статистику
func GetStats() Stats {
	return Stats{
		ActiveConnections: activeConnsGlobal,
		TotalRequests:     totalRequests,
		TotalErrors:       totalErrors,
	}
}

var activeConnsGlobal int32

// TrackConnection трекает соединение для статистики
func TrackConnection(open bool) {
	if open {
		atomic.AddInt32(&activeConnsGlobal, 1)
	} else {
		atomic.AddInt32(&activeConnsGlobal, -1)
	}
}
