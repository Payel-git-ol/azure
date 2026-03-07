// Package azurewebsockets - ультра-быстрые WebSocket для Azure фреймворка
// Zero-copy операции, минимум аллокаций, пулы объектов, оптимизации CPU
package azurewebsockets

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	azure "github.com/Payel-git-ol/azure"
	"github.com/Payel-git-ol/azure/ultrahttp"
)

// === КОНСТАНТЫ ===

const (
	// WebSocket опcodes
	OpContinuation = 0x0
	OpText         = 0x1
	OpBinary       = 0x2
	OpClose        = 0x8
	OpPing         = 0x9
	OpPong         = 0xA

	// Маски битов
	maskBit    = 0x80
	opcodeMask = 0x0F

	// Максимальный размер заголовка фрейма
	maxHeaderSize = 14

	// WebSocket GUID для handshake
	wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

	// Размер буфера по умолчанию (8KB)
	defaultBufSize = 8192

	// Размер пула буферов
	poolSize = 256
)

// === БЫСТРЫЕ КОНВЕРСИИ ===

// b2s - byte to string без аллокаций
//
//go:nosplit
func b2s(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// s2b - string to byte без аллокаций
//
//go:nosplit
func s2b(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(unsafe.Pointer(&s)))
}

// === SPINLOCK ===

// spinlock - примитив блокировки для коротких критических секций
type spinlock uint32

func (sl *spinlock) Lock() {
	for i := 0; i < 1000; i++ {
		if atomic.CompareAndSwapUint32((*uint32)(sl), 0, 1) {
			return
		}
		runtime.Gosched()
	}
	// Fallback на медленный путь
	for !atomic.CompareAndSwapUint32((*uint32)(sl), 0, 1) {
		runtime.Gosched()
	}
}

func (sl *spinlock) Unlock() {
	atomic.StoreUint32((*uint32)(sl), 0)
}

// === WEBSOCKET CONN ===

// Conn WebSocket соединение
type Conn struct {
	conn     *ultrahttp.Context
	buf      []byte
	maskKey  [4]byte
	isClient bool
	closed   uint32 // atomic flag
}

// === POOL'Ы ===

// connPool - пул Conn объектов
var connPool = sync.Pool{
	New: func() interface{} {
		return &Conn{
			buf: make([]byte, defaultBufSize),
		}
	},
}

// bufPool - пул буферов для Write операций
var bufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, defaultBufSize)
		return &buf
	},
}

// headerBufPool - пул для заголовков
var headerBufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, maxHeaderSize)
		return &buf
	},
}

// GetConn получает соединение из пула
//
//go:noinline
func GetConn(c *ultrahttp.Context) *Conn {
	ws := connPool.Get().(*Conn)
	ws.conn = c
	ws.isClient = false
	atomic.StoreUint32(&ws.closed, 0)
	return ws
}

// PutConn возвращает соединение в пул
//
//go:noinline
func PutConn(ws *Conn) {
	// Очищаем ссылки для GC
	ws.conn = nil
	connPool.Put(ws)
}

// === WEBSOCKET HANDSHAKE ===

// Upgrade проверяет WebSocket handshake и апгрейдит соединение
//
//go:noinline
func Upgrade(c *ultrahttp.Context) (*Conn, error) {
	// Проверяем заголовки
	upgrade := c.GetHeader("Upgrade")
	if upgrade == "" {
		return nil, ErrNotWebSocket
	}

	// Быстрая проверка на "websocket" без аллокаций
	upgradeBytes := s2b(upgrade)
	if !bytesEqualFold(upgradeBytes, []byte("websocket")) {
		return nil, ErrNotWebSocket
	}

	// Проверяем Connection header
	connection := c.GetHeader("Connection")
	connBytes := s2b(connection)
	if !containsIgnoreCase(connBytes, []byte("upgrade")) {
		return nil, ErrBadConnection
	}

	// Получаем Sec-WebSocket-Key
	key := c.GetHeader("Sec-WebSocket-Key")
	if key == "" {
		return nil, ErrBadKey
	}

	// Генерируем accept key
	accept := computeAcceptKey(key)

	// Отправляем ответ
	c.SetStatus(101, "Switching Protocols")
	c.SetHeader("Upgrade", "websocket")
	c.SetHeader("Connection", "Upgrade")
	c.SetHeader("Sec-WebSocket-Accept", accept)

	// Для ultrahttp просто отправляем ответ
	ws := GetConn(c)
	return ws, nil
}

// computeAcceptKey вычисляет accept key
//
//go:noinline
func computeAcceptKey(key string) string {
	// Используем stack buffer для ключа
	var keyBuf [100]byte
	copy(keyBuf[:], key)
	copy(keyBuf[len(key):], wsGUID)

	h := sha1.New()
	h.Write(s2b(key))
	h.Write([]byte(wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// bytesEqualFold - быстрое сравнение без учёта регистра
//
//go:nosplit
func bytesEqualFold(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		ca := a[i]
		cb := b[i]
		// Конвертируем в lowercase без ветвлений
		ca |= 0x20
		cb |= 0x20
		if ca != cb {
			return false
		}
	}
	return true
}

// containsIgnoreCase - проверка подстроки без учёта регистра
//
//go:noinline
func containsIgnoreCase(haystack, needle []byte) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if bytesEqualFold(haystack[i:i+len(needle)], needle) {
			return true
		}
	}
	return false
}

// === ОШИБКИ ===

var (
	ErrNotWebSocket    = &wsError{"not a websocket request"}
	ErrBadConnection   = &wsError{"bad connection header"}
	ErrBadKey          = &wsError{"missing or bad websocket key"}
	ErrInvalidFrame    = &wsError{"invalid websocket frame"}
	ErrClosed          = &wsError{"connection closed"}
	ErrBadOpcode       = &wsError{"bad opcode"}
	ErrControlTooLarge = &wsError{"control frame too large"}
)

type wsError struct {
	msg string
}

func (e *wsError) Error() string {
	return e.msg
}

// === ЧТЕНИЕ ФРЕЙМОВ ===

// ReadMessage читает WebSocket сообщение
// Возвращает opcode и данные (zero-copy когда возможно)
//
//go:noinline
func (ws *Conn) ReadMessage() (opcode int, data []byte, err error) {
	if atomic.LoadUint32(&ws.closed) != 0 {
		return 0, nil, ErrClosed
	}

	conn := ws.conn.GetConn()
	if conn == nil {
		return 0, nil, ErrClosed
	}

	// Читаем первые 2 байта заголовка
	header := ws.buf[:2]
	_, err = io.ReadFull(conn, header)
	if err != nil {
		return 0, nil, err
	}

	// Парсим заголовок
	opcode = int(header[0] & opcodeMask)
	masked := (header[1] & maskBit) != 0
	payloadLen := int(header[1] & 0x7F)

	// Проверяем opcode
	if opcode < OpContinuation || opcode > OpPong {
		return 0, nil, ErrBadOpcode
	}

	// Extended payload length
	if payloadLen == 126 {
		_, err = io.ReadFull(conn, ws.buf[:2])
		if err != nil {
			return 0, nil, err
		}
		payloadLen = int(binary.BigEndian.Uint16(ws.buf[:2]))
	} else if payloadLen == 127 {
		_, err = io.ReadFull(conn, ws.buf[:8])
		if err != nil {
			return 0, nil, err
		}
		payloadLen = int(binary.BigEndian.Uint64(ws.buf[:8]))
	}

	// Маска
	var maskKey [4]byte
	if masked {
		_, err = io.ReadFull(conn, ws.buf[:4])
		if err != nil {
			return 0, nil, err
		}
		maskKey = [4]byte{ws.buf[0], ws.buf[1], ws.buf[2], ws.buf[3]}
	}

	// Читаем payload - используем буфер из пула если большой
	if payloadLen > defaultBufSize {
		data = make([]byte, payloadLen)
	} else {
		data = ws.buf[:payloadLen]
	}

	if payloadLen > 0 {
		_, err = io.ReadFull(conn, data)
		if err != nil {
			return 0, nil, err
		}

		// Демаскируем - оптимизированная версия
		if masked {
			xorMaskFast(data, maskKey)
		}
	}

	// Обрабатываем control фреймы
	if opcode >= OpClose {
		ws.handleControl(opcode, data)
		return ws.ReadMessage()
	}

	return opcode, data, nil
}

// xorMaskFast - оптимизированная XOR маска с unroll
//
//go:noinline
func xorMaskFast(data []byte, mask [4]byte) {
	// Unroll loop для 4 байт за раз
	n := len(data)
	i := 0

	// Обрабатываем полные 4-байтные блоки
	for ; i <= n-4; i += 4 {
		data[i] ^= mask[0]
		data[i+1] ^= mask[1]
		data[i+2] ^= mask[2]
		data[i+3] ^= mask[3]
	}

	// Остаток
	for ; i < n; i++ {
		data[i] ^= mask[i&3]
	}
}

// handleControl обрабатывает control фреймы
func (ws *Conn) handleControl(opcode int, data []byte) {
	switch opcode {
	case OpClose:
		atomic.StoreUint32(&ws.closed, 1)
	case OpPing:
		ws.WriteMessage(OpPong, data)
	}
}

// === ЗАПИСЬ ФРЕЙМОВ ===

// WriteMessage записывает WebSocket сообщение
//
//go:noinline
func (ws *Conn) WriteMessage(opcode int, data []byte) error {
	if atomic.LoadUint32(&ws.closed) != 0 {
		return ErrClosed
	}

	conn := ws.conn.GetConn()
	if conn == nil {
		return ErrClosed
	}

	// Берём буфер из пула
	headerBuf := headerBufPool.Get().(*[]byte)
	defer headerBufPool.Put(headerBuf)

	pos := 0

	// Byte 0: FIN + opcode
	(*headerBuf)[pos] = byte(0x80 | opcode)
	pos++

	// Byte 1: маска + длина
	length := len(data)

	if length <= 125 {
		(*headerBuf)[pos] = byte(length)
		pos++
	} else if length <= 65535 {
		(*headerBuf)[pos] = 126
		pos++
		binary.BigEndian.PutUint16((*headerBuf)[pos:], uint16(length))
		pos += 2
	} else {
		(*headerBuf)[pos] = 127
		pos++
		binary.BigEndian.PutUint64((*headerBuf)[pos:], uint64(length))
		pos += 8
	}

	// Пишем заголовок
	_, err := conn.Write((*headerBuf)[:pos])
	if err != nil {
		return err
	}

	// Пишем данные
	if length > 0 {
		_, err = conn.Write(data)
	}

	runtime.KeepAlive(data)
	return err
}

// WriteText отправляет текстовое сообщение
func (ws *Conn) WriteText(data []byte) error {
	return ws.WriteMessage(OpText, data)
}

// WriteBinary отправляет бинарное сообщение
func (ws *Conn) WriteBinary(data []byte) error {
	return ws.WriteMessage(OpBinary, data)
}

// WriteJSON отправляет JSON сообщение
func (ws *Conn) WriteJSON(v interface{}) error {
	data := fastMarshalJSON(v)
	return ws.WriteMessage(OpText, data)
}

// fastMarshalJSON - быстрая сериализация JSON
func fastMarshalJSON(v interface{}) []byte {
	if m, ok := v.(azure.M); ok {
		// Конвертируем azure.M в ultrahttp.M
		um := make(ultrahttp.M, len(m))
		for k, v := range m {
			um[k] = v
		}
		return ultrahttp.FastMarshalM(um)
	}
	buf, _ := ultrahttp.MarshalJSON(v)
	return buf
}

// Close закрывает соединение
func (ws *Conn) Close() error {
	if atomic.SwapUint32(&ws.closed, 1) != 0 {
		return nil
	}

	conn := ws.conn.GetConn()
	if conn == nil {
		return nil
	}

	// Отправляем close фрейм
	closeFrame := []byte{0x88, 0x02, 0x03, 0xE8} // FIN+Close, 2 bytes, 1000 Normal Closure
	conn.Write(closeFrame)
	return conn.Close()
}

// IsClosed проверяет закрыто ли соединение
func (ws *Conn) IsClosed() bool {
	return atomic.LoadUint32(&ws.closed) != 0
}

// === AZURE ИНТЕГРАЦИЯ ===

// Handler тип обработчика WebSocket сообщений
type Handler func(ws *Conn, opcode int, data []byte)

// Middleware создаёт Azure middleware для WebSocket
func Middleware(handler Handler) azure.Middleware {
	return func(c *azure.Context, next ultrahttp.RouteHandler) {
		// Проверяем WebSocket upgrade
		upgrade := c.GetHeader("Upgrade")
		if upgrade == "" {
			next(c.GetUltra())
			return
		}

		// Получаем net.Conn напрямую
		conn := c.GetUltra().GetConn()
		if conn == nil {
			c.SetStatus(500, "Internal Server Error")
			c.Json(azure.M{
				"error": "Cannot get underlying connection",
			})
			return
		}

		// Генерируем accept key
		key := c.GetHeader("Sec-WebSocket-Key")
		accept := computeAcceptKey(key)

		// Отправляем WebSocket handshake ответ напрямую в conn
		handshake := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + accept + "\r\n" +
			"\r\n"

		_, err := conn.Write([]byte(handshake))
		if err != nil {
			return
		}

		// Создаём Conn для работы с WebSocket
		ws := &Conn{
			conn:     c.GetUltra(),
			buf:      make([]byte, defaultBufSize),
			isClient: false,
			closed:   0,
		}

		// Вызываем хендлер в горутине
		go wsHandler(ws, handler)

		// Важно: не возвращаем управление фреймворку
		select {}
	}
}

// HandlerFunc создаёт хендлер для использования с a.Get()/a.Post() и т.д.
func HandlerFunc(handler Handler) func(c *azure.Context) {
	return func(c *azure.Context) {
		// Проверяем WebSocket upgrade
		upgrade := c.GetHeader("Upgrade")
		if upgrade == "" {
			c.SetStatus(400, "Bad Request")
			c.Json(azure.M{
				"error": "Not a WebSocket request",
			})
			return
		}

		// Получаем net.Conn напрямую
		conn := c.GetUltra().GetConn()
		if conn == nil {
			c.SetStatus(500, "Internal Server Error")
			c.Json(azure.M{
				"error": "Cannot get underlying connection",
			})
			return
		}

		// Генерируем accept key
		key := c.GetHeader("Sec-WebSocket-Key")
		accept := computeAcceptKey(key)

		// Отправляем WebSocket handshake ответ напрямую в conn
		handshake := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"Sec-WebSocket-Accept: " + accept + "\r\n" +
			"\r\n"

		_, err := conn.Write([]byte(handshake))
		if err != nil {
			return
		}

		// Создаём Conn для работы с WebSocket
		ws := &Conn{
			conn:     c.GetUltra(),
			buf:      make([]byte, defaultBufSize),
			isClient: false,
			closed:   0,
		}

		// Вызываем хендлер в горутине
		go wsHandler(ws, handler)

		// Важно: не возвращаем управление фреймворку
		select {}
	}
}

// wsHandler обрабатывает сообщения в цикле
//
//go:noinline
func wsHandler(ws *Conn, handler Handler) {
	for {
		opcode, data, err := ws.ReadMessage()
		if err != nil {
			ws.Close()
			return
		}
		handler(ws, opcode, data)
	}
}
