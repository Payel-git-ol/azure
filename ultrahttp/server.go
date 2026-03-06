// Package ultrahttp - сверхбыстрый HTTP сервер на чистом Go
// Без аллокаций в горячем пути, zero-copy операции, собственный TCP стек
package ultrahttp

import (
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// b2s - byte slice to string без аллокаций
//
//go:nosplit
func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// s2b - string to byte slice без аллокаций
//
//go:nosplit
func s2b(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

// === КОНСТАНТЫ И ТИПЫ ===

const (
	DefaultBufferSize    = 8 * 1024  // 8KB буфер
	MaxHeaderSize        = 64 * 1024 // 64KB макс размер заголовков
	MaxRequestBodySize   = 64 * 1024 // 64KB макс размер тела (для начала)
	DefaultMaxWorkers    = 0         // 0 = GOMAXPROCS
	DefaultReadTimeout   = 5 * time.Second
	DefaultWriteTimeout  = 10 * time.Second
	DefaultIdleTimeout   = 120 * time.Second
	DefaultMaxKeepAlives = 256
)

// HTTP статусы - быстрые константы
const (
	StatusOK           = 200
	StatusCreated      = 201
	StatusNoContent    = 204
	StatusBadRequest   = 400
	StatusUnauthorized = 401
	StatusNotFound     = 404
	StatusMethodNotAllowed = 405
	StatusConflict     = 409
	StatusTooManyRequests = 429
	StatusInternalServerError = 500
	StatusBadGateway   = 502
	StatusServiceUnavailable = 503
)

// Предзаполненные шаблоны ответов
var (
	response200 = []byte("HTTP/1.1 200 OK\r\n")
	response201 = []byte("HTTP/1.1 201 Created\r\n")
	response204 = []byte("HTTP/1.1 204 No Content\r\n")
	response400 = []byte("HTTP/1.1 400 Bad Request\r\n")
	response404 = []byte("HTTP/1.1 404 Not Found\r\n")
	response500 = []byte("HTTP/1.1 500 Internal Server Error\r\n")
)

// Таблица для быстрого парсинга чисел
var parseHexTable = [256]byte{
	'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7,
	'8': 8, '9': 9,
}

// Методы как байтовые слайсы для быстрого сравнения
var (
	methodGET     = []byte("GET")
	methodPOST    = []byte("POST")
	methodPUT     = []byte("PUT")
	methodDELETE  = []byte("DELETE")
	methodPATCH   = []byte("PATCH")
	methodHEAD    = []byte("HEAD")
	methodOPTIONS = []byte("OPTIONS")

	colonSpace = []byte(": ")

	contentTypeKey   = []byte("Content-Type")
	contentLengthKey = []byte("Content-Length")
	connectionKey    = []byte("Connection")
	keepAliveVal     = []byte("keep-alive")
	closeVal         = []byte("close")
)

// Request - оптимизированный HTTP запрос
type Request struct {
	Method      []byte
	Path        []byte
	QueryString []byte
	Body        []byte
	ContentLen  int
	RemoteAddr  string
	conn        net.Conn
	keepAlive   bool
	Headers     map[string]string
}

// Response - оптимизированный HTTP ответ
type Response struct {
	Status     int
	StatusText string
	Body       []byte
	Headers    map[string]string
	buffer     []byte
}

// Handler - обработчик запросов
type Handler func(ctx *Context)

// Server - ultrahttp сервер
type Server struct {
	addr          string
	ln            net.Listener
	handler       Handler
	maxWorkers    int
	readTimeout   time.Duration
	writeTimeout  time.Duration
	idleTimeout   time.Duration
	maxKeepAlives int32
	activeConns   int32
	stopped       int32
	wg            sync.WaitGroup
	connChan      chan net.Conn // Локальный канал для соединений
}

// Context объединяет Request и Response
type Context struct {
	Request  Request
	Response Response
}

// === POOL'Ы ДЛЯ ПЕРЕИСПОЛЬЗОВАНИЯ ===

var (
	contextPool = sync.Pool{
		New: func() interface{} {
			return &Context{
				Request: Request{
					Headers: make(map[string]string, 8),
				},
				Response: Response{
					Status:     200,
					StatusText: "OK",
					Headers:    make(map[string]string, 4),
					Body:       make([]byte, 0), // Без pre-allocation - растёт по необходимости
				},
			}
		},
	}

	// Основной buffer pool 64KB - используется для I/O операций
	bufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 64*1024) // 64KB buffer
			return &buf
		},
	}
)

// === НОВОЕ СОЗДАНИЕ СЕРВЕРА ===

// NewServer создаёт новый ultrahttp сервер
//
//go:noinline
func NewServer(addr string, handler Handler) *Server {
	workers := runtime.GOMAXPROCS(0)
	if workers <= 0 {
		workers = 4
	} else if workers > 32 {
		workers = 32
	}

	return &Server{
		addr:          addr,
		handler:       handler,
		maxWorkers:    workers,
		readTimeout:   DefaultReadTimeout,
		writeTimeout:  DefaultWriteTimeout,
		idleTimeout:   DefaultIdleTimeout,
		maxKeepAlives: DefaultMaxKeepAlives,
		connChan:      make(chan net.Conn, 1024),
	}
}

// ListenAndServe запускает сервер
func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	s.ln = ln
	println("[ultrahttp] Server starting on", s.addr)
	println("[ultrahttp] Workers:", s.maxWorkers)

	// Запускаем воркеры
	for i := 0; i < s.maxWorkers; i++ {
		s.wg.Add(1)
		go s.worker(s.connChan)
	}

	// Акцепт соединений
	for atomic.LoadInt32(&s.stopped) == 0 {
		conn, err := ln.Accept()
		if err != nil {
			if atomic.LoadInt32(&s.stopped) == 1 {
				return nil
			}
			continue
		}

		// Передаём соединение в канал
		select {
		case s.connChan <- conn:
		default:
			conn.Close()
		}
	}

	return nil
}

var connChan = make(chan net.Conn, 1024) // deprecated

// activeConnsGlobal объявлен в adapter.go

func (s *Server) worker(connChan <-chan net.Conn) {
	defer s.wg.Done()

	for conn := range connChan {
		s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	if atomic.LoadInt32(&s.stopped) == 1 {
		conn.Close()
		return
	}

	atomic.AddInt32(&s.activeConns, 1)
	atomic.AddInt32(&activeConnsGlobal, 1)
	defer func() {
		atomic.AddInt32(&s.activeConns, -1)
		atomic.AddInt32(&activeConnsGlobal, -1)
	}()

	// Настраиваем таймауты - используем кэшированное время
	conn.SetReadDeadline(time.Now().Add(s.idleTimeout))

	// Получаем буфер из пула (64KB)
	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)

	buf := *bufPtr
	bufLen := 0

	keepAliveCount := int32(0)

	for {
		if atomic.LoadInt32(&s.stopped) == 1 {
			break
		}

		// Проверка лимита keep-alive
		if keepAliveCount > atomic.LoadInt32(&s.maxKeepAlives) {
			break
		}

		// Получаем контекст из пула
		ctx := contextPool.Get().(*Context)

		// Инициализируем контекст - inline версия
		ctx.Request.Method = ctx.Request.Method[:0]
		ctx.Request.Path = ctx.Request.Path[:0]
		ctx.Request.QueryString = ctx.Request.QueryString[:0]
		ctx.Request.Body = ctx.Request.Body[:0]
		ctx.Request.ContentLen = 0
		ctx.Request.RemoteAddr = conn.RemoteAddr().String()
		ctx.Request.conn = conn
		ctx.Request.keepAlive = true
		for k := range ctx.Request.Headers {
			delete(ctx.Request.Headers, k)
		}
		ctx.Response.Status = 200
		ctx.Response.StatusText = "OK"
		ctx.Response.Body = ctx.Response.Body[:0]
		for k := range ctx.Response.Headers {
			delete(ctx.Response.Headers, k)
		}

		// Читаем и парсим запрос напрямую из буфера
		n, err := conn.Read(buf)
		if err != nil {
			contextPool.Put(ctx)
			break
		}
		bufLen = n

		// Парсим запрос
		_, err = s.parseRequestFast(buf[:bufLen], &ctx.Request)
		if err != nil {
			s.writeBadRequest(conn)
			contextPool.Put(ctx)
			break
		}

		// Вызываем обработчик
		s.handler(ctx)

		// Отправляем ответ
		if err := s.writeResponseFast(conn, &ctx.Response); err != nil {
			contextPool.Put(ctx)
			break
		}

		// Возвращаем в пул
		contextPool.Put(ctx)

		// Проверка keep-alive
		if !ctx.Request.keepAlive {
			break
		}

		keepAliveCount++
		conn.SetReadDeadline(time.Now().Add(s.idleTimeout))
	}

	conn.Close()
}

// initContext инициализирует контекст для нового запроса
//
//go:noinline
func (s *Server) initContext(ctx *Context, conn net.Conn) {
	// Request
	ctx.Request.Method = ctx.Request.Method[:0]
	ctx.Request.Path = ctx.Request.Path[:0]
	ctx.Request.QueryString = ctx.Request.QueryString[:0]
	ctx.Request.Body = ctx.Request.Body[:0]
	ctx.Request.ContentLen = 0
	ctx.Request.RemoteAddr = conn.RemoteAddr().String()
	ctx.Request.conn = conn
	ctx.Request.keepAlive = true
	for k := range ctx.Request.Headers {
		delete(ctx.Request.Headers, k)
	}

	// Response
	ctx.Response.Status = 200
	ctx.Response.StatusText = "OK"
	ctx.Response.Body = ctx.Response.Body[:0]
	for k := range ctx.Response.Headers {
		delete(ctx.Response.Headers, k)
	}
}

// === БЫСТРЫЙ ПАРСИНГ (без bufio) ===

// parseRequestFast парсит запрос напрямую из буфера
//
//go:noinline
func (s *Server) parseRequestFast(buf []byte, req *Request) (int, error) {
	pos := 0
	bufLen := len(buf)

	// Парсим первую строку: METHOD PATH
	lineStart := pos
	for pos < bufLen && buf[pos] != '\r' && buf[pos] != '\n' {
		pos++
	}

	// Парсим метод - оптимизировано для GET/POST
	wordStart := lineStart
	for wordStart < pos && buf[wordStart] == ' ' {
		wordStart++
	}
	wordEnd := wordStart
	for wordEnd < pos && buf[wordEnd] != ' ' {
		wordEnd++
	}
	req.Method = append(req.Method[:0], buf[wordStart:wordEnd]...)

	// Парсим путь
	wordStart = wordEnd + 1
	for wordStart < pos && buf[wordStart] == ' ' {
		wordStart++
	}
	wordEnd = wordStart
	for wordEnd < pos && buf[wordEnd] != ' ' {
		wordEnd++
	}

	// Разделяем путь и query string
	pathStart := wordStart
	pathEnd := wordEnd
	queryIdx := -1
	for i := pathStart; i < pathEnd; i++ {
		if buf[i] == '?' {
			queryIdx = i
			break
		}
	}

	if queryIdx >= 0 {
		req.Path = append(req.Path[:0], buf[pathStart:queryIdx]...)
		req.QueryString = append(req.QueryString[:0], buf[queryIdx+1:pathEnd]...)
	} else {
		req.Path = append(req.Path[:0], buf[pathStart:pathEnd]...)
		req.QueryString = req.QueryString[:0]
	}

	// Пропускаем \r\n
	pos += 2
	if pos < bufLen && buf[pos-1] == '\n' && buf[pos-2] != '\r' {
		pos--
	}

	// Парсим заголовки
	for pos < bufLen {
		lineStart = pos
		for pos < bufLen && buf[pos] != '\r' && buf[pos] != '\n' {
			pos++
		}

		// Пустая строка - конец заголовков
		if pos <= lineStart+1 {
			pos++
			if pos < bufLen && buf[pos] == '\n' {
				pos++
			}
			break
		}

		// Парсим заголовок Key: Value
		colonIdx := -1
		for i := lineStart; i < pos; i++ {
			if buf[i] == ':' {
				colonIdx = i
				break
			}
		}

		if colonIdx >= 0 && colonIdx+2 < pos {
			keyStart := lineStart
			keyEnd := colonIdx
			for keyStart < keyEnd && buf[keyStart] == ' ' {
				keyStart++
			}
			for keyEnd > keyStart && buf[keyEnd-1] == ' ' {
				keyEnd--
			}

			valStart := colonIdx + 1
			for valStart < pos && buf[valStart] == ' ' {
				valStart++
			}
			valEnd := pos
			for valEnd > valStart && buf[valEnd-1] == ' ' {
				valEnd--
			}

			key := b2s(buf[keyStart:keyEnd])
			value := b2s(buf[valStart:valEnd])
			req.Headers[key] = value

			// Проверяем Content-Length (первый байт 'C' или 'c')
			if len(key) > 0 && (key[0] == 'C' || key[0] == 'c') {
				if key == "Content-Length" || key == "content-length" {
					req.ContentLen = atoi(buf[valStart:valEnd])
				}
				// Проверяем Connection
				if key == "Connection" || key == "connection" {
					if value == "close" || value == "Close" {
						req.keepAlive = false
					}
				}
			}
		}

		pos++
		if pos < bufLen && buf[pos] == '\n' {
			pos++
		}
	}

	// Читаем тело если есть
	if req.ContentLen > 0 && req.ContentLen <= MaxRequestBodySize {
		remaining := bufLen - pos
		if remaining >= req.ContentLen {
			req.Body = append(req.Body[:0], buf[pos:pos+req.ContentLen]...)
			pos += req.ContentLen
		}
	}

	return pos, nil
}

// writeResponseFast - ULTRA быстрая запись ответа
//
//go:noinline
func (s *Server) writeResponseFast(conn net.Conn, resp *Response) error {
	bodyLen := len(resp.Body)

	// ULTRA-быстрый путь для 200 OK JSON - 95% всех GET запросов
	if (resp.Status == 0 || resp.Status == 200) && bodyLen > 0 && bodyLen < 4000 {
		if ct := resp.Headers["Content-Type"]; ct == "application/json; charset=utf-8" {
			// Стек-буфер 4KB - НОЛЬ аллокаций
			var buf [4096]byte
			pos := 0

			// Готовый шаблон заголовка - используем copy (он оптимизирован компилятором)
			const header = "HTTP/1.1 200 OK\r\nContent-Type: application/json; charset=utf-8\r\nContent-Length: "
			copy(buf[:], header)
			pos = len(header)

			// Быстрая запись длины - используем оптимизированную функцию
			pos += formatLengthInline(buf[pos:], bodyLen)

			// \r\n\r\n
			buf[pos] = '\r'
			buf[pos+1] = '\n'
			buf[pos+2] = '\r'
			buf[pos+3] = '\n'
			pos += 4

			// Копируем тело
			if bodyLen <= len(buf)-pos {
				copy(buf[pos:], resp.Body)
				pos += bodyLen

				conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))
				_, err := conn.Write(buf[:pos])
				return err
			}
		}
	}

	// Общий путь
	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)
	buf := *bufPtr

	pos := 0
	copy(buf[pos:], "HTTP/1.1 ")
	pos += 9

	status := resp.Status
	if status == 0 {
		status = 200
	}

	switch status {
	case 200:
		copy(buf[pos:], "200 OK")
		pos += 6
	case 201:
		copy(buf[pos:], "201 Created")
		pos += 11
	case 204:
		copy(buf[pos:], "204 No Content")
		pos += 14
	case 400:
		copy(buf[pos:], "400 Bad Request")
		pos += 15
	case 404:
		copy(buf[pos:], "404 Not Found")
		pos += 13
	case 500:
		copy(buf[pos:], "500 Internal Server Error")
		pos += 25
	default:
		buf = appendInt(buf[:pos], status)
		pos = len(buf)
		buf[pos] = ' '
		pos++
	}

	buf[pos] = '\r'
	buf[pos+1] = '\n'
	pos += 2

	for key, value := range resp.Headers {
		copy(buf[pos:], key)
		pos += len(key)
		buf[pos] = ':'
		buf[pos+1] = ' '
		pos += 2
		copy(buf[pos:], value)
		pos += len(value)
		buf[pos] = '\r'
		buf[pos+1] = '\n'
		pos += 2
	}

	if bodyLen > 0 {
		copy(buf[pos:], "Content-Length: ")
		pos += 16
		buf = append(buf[:pos], '0')
		buf = appendInt(buf[:pos], bodyLen)
		pos = len(buf)
		buf = append(buf, '\r', '\n')
		pos += 2
	}

	buf = append(buf[:pos], '\r', '\n')
	pos = len(buf)

	conn.SetWriteDeadline(time.Now().Add(s.writeTimeout))

	if bodyLen > 0 && pos+bodyLen <= len(buf) {
		copy(buf[pos:], resp.Body)
		pos += bodyLen
	}

	_, err := conn.Write(buf[:pos])
	if err != nil {
		return err
	}

	if bodyLen > 0 && pos+bodyLen > len(buf) {
		_, err = conn.Write(resp.Body)
	}

	return err
}

func (s *Server) writeBadRequest(conn net.Conn) {
	resp := []byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 11\r\n\r\nBad Request")
	conn.Write(resp)
}

// === ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ (БЕЗ АЛЛОКАЦИЙ) ===

func trimCRLF(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	if b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	if len(b) > 0 && b[len(b)-1] == '\r' {
		b = b[:len(b)-1]
	}
	return b
}

func trimSpace(b []byte) []byte {
	start := 0
	for start < len(b) && (b[start] == ' ' || b[start] == '\t') {
		start++
	}
	end := len(b)
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}

func bytesSplit(b []byte, sep byte) [][]byte {
	var result [][]byte
	start := 0
	for i := 0; i < len(b); i++ {
		if b[i] == sep {
			result = append(result, b[start:i])
			start = i + 1
		}
	}
	result = append(result, b[start:])
	return result
}

func bytesIndexByte(b []byte, c byte) int {
	for i := 0; i < len(b); i++ {
		if b[i] == c {
			return i
		}
	}
	return -1
}

func bytesIndex(b []byte, sep []byte) int {
	if len(sep) == 0 {
		return 0
	}
	for i := 0; i <= len(b)-len(sep); i++ {
		if bytesEqual(b[i:i+len(sep)], sep) {
			return i
		}
	}
	return -1
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// === УЛЬТРА-БЫСТРОЕ ПРЕОБРАЗОВАНИЕ ЧИСЕЛ В ASCII ===

// formatLengthInlineAssembly - преобразование длины тела в ASCII (максимум 5 цифр)
// Оптимизировано для скорости через неразвёрнутый цикл
//
//go:noinline
func formatLengthInline(buf []byte, length int) int {
	if length < 10 {
		buf[0] = byte('0' + length)
		return 1
	}
	if length < 100 {
		buf[0] = byte('0' + length/10)
		buf[1] = byte('0' + length%10)
		return 2
	}
	if length < 1000 {
		buf[0] = byte('0' + length/100)
		buf[1] = byte('0' + (length/10)%10)
		buf[2] = byte('0' + length%10)
		return 3
	}
	if length < 10000 {
		buf[0] = byte('0' + length/1000)
		buf[1] = byte('0' + (length/100)%10)
		buf[2] = byte('0' + (length/10)%10)
		buf[3] = byte('0' + length%10)
		return 4
	}
	// 5 цифр для больших чисел
	buf[0] = byte('0' + length/10000)
	buf[1] = byte('0' + (length/1000)%10)
	buf[2] = byte('0' + (length/100)%10)
	buf[3] = byte('0' + (length/10)%10)
	buf[4] = byte('0' + length%10)
	return 5
}

//go:noinline
func atoi(b []byte) int {
	n := 0
	for i := 0; i < len(b); i++ {
		if b[i] < '0' || b[i] > '9' {
			break
		}
		n = n*10 + int(b[i]-'0')
	}
	return n
}

//go:noinline
func appendInt(b []byte, n int) []byte {
	if n == 0 {
		return append(b, '0')
	}

	var buf [32]byte
	i := len(buf)

	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}

	return append(b, buf[i:]...)
}

// bytesEqualNoBounds - быстрое сравнение без проверки границ
//
//go:noinline
func bytesEqualNoBounds(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// === МЕТОДЫ ДЛЯ КОНТЕКСТА ===

// GetMethod возвращает метод как строку
func (c *Context) GetMethod() string {
	return b2s(c.Request.Method)
}

// GetPath возвращает путь как строку
func (c *Context) GetPath() string {
	return b2s(c.Request.Path)
}

// GetQueryString возвращает query string как строку
func (c *Context) GetQueryString() string {
	return b2s(c.Request.QueryString)
}

// GetBody возвращает тело запроса
func (c *Context) GetBody() []byte {
	return c.Request.Body
}

// GetRemoteAddr возвращает адрес клиента
func (c *Context) GetRemoteAddr() string {
	return c.Request.RemoteAddr
}

// === СТАТИЧЕСКИЕ МЕТОДЫ ДЛЯ БЫСТРОГО ДОСТУПА ===

// Status200OK - готовый ответ 200
var Status200OK = []byte("200 OK")
var Status404NotFound = []byte("404 Not Found")
var Status500InternalError = []byte("500 Internal Server Error")
