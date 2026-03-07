# Azure WebSocket

Ультра-быстрые WebSocket для Azure фреймворка с минимальными аллокациями и zero-copy операциями.

## Особенности

- ⚡ **Zero-copy** операции где возможно
- 🔄 **Pool объектов** для переиспользования Conn
- 🚀 **Оптимизированный парсинг** handshake без аллокаций
- 📦 **Минимум GC давления** - буферы переиспользуются
- 🔧 **Простая интеграция** с Azure middleware

## Быстрый старт

```go
package main

import (
    "github.com/Payel-git-ol/azure"
    "github.com/Payel-git-ol/azure/azurewebsockets"
    "log"
)

func main() {
    a := azure.New()

    // WebSocket хендлер
    wsHandler := func(ws *azurewebsockets.Conn, opcode int, data []byte) {
        // Эхо - отправляем полученное обратно
        ws.WriteMessage(opcode, data)
        
        // Или JSON
        // ws.WriteJSON(azure.M{"received": string(data)})
    }

    // Регистрируем WebSocket endpoint
    a.Get("/ws", azurewebsockets.Middleware(wsHandler))

    log.Println("Server starting on :7070")
    a.Run(":7070")
}
```

## Использование

### Чтение сообщений

```go
wsHandler := func(ws *azurewebsockets.Conn, opcode int, data []byte) {
    switch opcode {
    case azurewebsockets.OpText:
        log.Printf("Текст: %s", data)
        
    case azurewebsockets.OpBinary:
        log.Printf("Бинарные данные: %d байт", len(data))
        
    case azurewebsockets.OpClose:
        log.Println("Клиент закрыл соединение")
        ws.Close()
    }
}
```

### Отправка сообщений

```go
// Текстовое сообщение
ws.WriteText([]byte("Hello, WebSocket!"))

// Бинарное сообщение
ws.WriteBinary(binaryData)

// JSON
ws.WriteJSON(azure.M{
    "type": "message",
    "data": "Hello from server",
})

// Закрыть соединение
ws.Close()
```

## Продвинутые примеры

### Чат комната

```go
var clients = make(map[*azurewebsockets.Conn]bool)
var clientsMu sync.Mutex

func chatHandler(ws *azurewebsockets.Conn, opcode int, data []byte) {
    if opcode == azurewebsockets.OpClose {
        clientsMu.Lock()
        delete(clients, ws)
        clientsMu.Unlock()
        return
    }

    // Рассылаем всем клиентам
    clientsMu.Lock()
    for client := range clients {
        client.WriteMessage(opcode, data)
    }
    clientsMu.Unlock()
}
```

### Ping/Pong для keepalive

```go
func pingPongHandler(ws *azurewebsockets.Conn, opcode int, data []byte) {
    switch opcode {
    case azurewebsockets.OpPing:
        ws.WriteMessage(azurewebsockets.OpPong, nil)
        
    case azurewebsockets.OpText:
        ws.WriteMessage(azurewebsockets.OpText, data)
    }
}
```

## API

### Функции

| Функция | Описание |
|---------|----------|
| `Upgrade(c *ultrahttp.Context) (*Conn, error)` | Апгрейд HTTP соединения в WebSocket |
| `Middleware(handler Handler) azure.Middleware` | Создаёт Azure middleware |
| `GetConn() *Conn` | Получить Conn из пула |
| `PutConn(ws *Conn)` | Вернуть Conn в пул |

### Conn методы

| Метод | Описание |
|-------|----------|
| `ReadMessage() (int, []byte, error)` | Читать сообщение (opcode, данные) |
| `WriteMessage(opcode int, data []byte) error` | Записать сообщение |
| `WriteText(data []byte) error` | Отправить текст |
| `WriteBinary(data []byte) error` | Отправить бинарные данные |
| `WriteJSON(v interface{}) error` | Отправить JSON |
| `Close() error` | Закрыть соединение |

### Константы

```go
const (
    OpContinuation = 0x0
    OpText         = 0x1
    OpBinary       = 0x2
    OpClose        = 0x8
    OpPing         = 0x9
    OpPong         = 0xA
)
```

## Производительность

Благодаря оптимизациям:
- **0 аллокаций** при handshake (кроме accept key)
- **1 аллокация** на сообщение (для payload)
- **Pool Conn** уменьшает давление на GC
- **Быстрый парсинг** заголовков без regex

## Лицензия

MIT
