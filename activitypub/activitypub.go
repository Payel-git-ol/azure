// Package activitypub - реализация ActivityPub протокола для Azure фреймворка
// Использует WebSocket для real-time федерации
// Спецификация: https://www.w3.org/TR/activitypub/
package activitypub

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Payel-git-ol/azure"
	"github.com/Payel-git-ol/azure/azurewebsockets"
	"github.com/Payel-git-ol/azure/ultrahttp"
)

// M - сокращение для map[string]interface{} (алиас на azure.M)
type M map[string]interface{}

// === КОНСТАНТЫ ACTIVITYPUB ===

// ActivityPub типы
const (
	// Activity типы
	ActivityCreate   = "Create"
	ActivityDelete   = "Delete"
	ActivityFollow   = "Follow"
	ActivityAccept   = "Accept"
	ActivityReject   = "Reject"
	ActivityUndo     = "Undo"
	ActivityUpdate   = "Update"
	ActivityLike     = "Like"
	ActivityAnnounce = "Announce"

	// Object типы
	ObjectNote    = "Note"
	ObjectArticle = "Article"
	ObjectImage   = "Image"
	ObjectVideo   = "Video"

	// Special collections
	CollectionOutbox    = "outbox"
	CollectionInbox     = "inbox"
	CollectionFollowing = "following"
	CollectionFollowers = "followers"
)

// === CONTEXT ===

// Context контекст ActivityPub сообщения
type Context struct {
	ws       *azurewebsockets.Conn
	opcode   int
	data     []byte
	object   *Object
	actor    *Actor
	response *Response
	azureCtx *azure.Context
	httpResp interface{} // HTTP ответ (если это не WebSocket)
}

// Object ActivityPub объект (упрощённая JSON-LD структура)
type Object struct {
	ID           string                 `json:"id,omitempty"`
	Type         string                 `json:"type"`
	Actor        string                 `json:"actor,omitempty"`
	Object       interface{}            `json:"object,omitempty"`
	Target       interface{}            `json:"target,omitempty"`
	To           []string               `json:"to,omitempty"`
	CC           []string               `json:"cc,omitempty"`
	Published    *time.Time             `json:"published,omitempty"`
	Updated      *time.Time             `json:"updated,omitempty"`
	AttributedTo string                 `json:"attributedTo,omitempty"`
	Content      string                 `json:"content,omitempty"`
	Name         string                 `json:"name,omitempty"`
	Summary      string                 `json:"summary,omitempty"`
	URL          string                 `json:"url,omitempty"`
	Icon         *Link                  `json:"icon,omitempty"`
	Image        *Link                  `json:"image,omitempty"`
	Replies      *Collection            `json:"replies,omitempty"`
	Attachments  []interface{}          `json:"attachment,omitempty"`
	Tag          []Tag                  `json:"tag,omitempty"`
	Sensitive    bool                   `json:"sensitive,omitempty"`
	Language     string                 `json:"language,omitempty"`
	Extra        map[string]interface{} `json:"-"` // Дополнительные JSON-LD поля
}

// Actor ActivityPub актёр
type Actor struct {
	ID                        string                 `json:"id"`
	Type                      string                 `json:"type"` // Person, Organization, Service, Application, Group
	Name                      string                 `json:"name"`
	PreferredUsername         string                 `json:"preferredUsername"`
	Inbox                     string                 `json:"inbox"`
	Outbox                    string                 `json:"outbox"`
	Following                 string                 `json:"following"`
	Followers                 string                 `json:"followers"`
	InboxItems                *Collection            `json:"inboxItems,omitempty"`
	OutboxItems               *Collection            `json:"outboxItems,omitempty"`
	PublicKey                 *PublicKey             `json:"publicKey,omitempty"`
	Endpoints                 *Endpoints             `json:"endpoints,omitempty"`
	Icon                      *Link                  `json:"icon,omitempty"`
	Image                     *Link                  `json:"image,omitempty"`
	Summary                   string                 `json:"summary"`
	URL                       string                 `json:"url"`
	ManuallyApprovesFollowers bool                   `json:"manuallyApprovesFollowers"`
	Discoverable              bool                   `json:"discoverable"`
	Published                 *time.Time             `json:"published"`
	Updated                   *time.Time             `json:"updated"`
	Extra                     map[string]interface{} `json:"-"`
	To                        []string
	CC                        []string
}

// Link ссылка
type Link struct {
	Type      string `json:"type"`
	MediaType string `json:"mediaType"`
	URL       string `json:"url"`
	Name      string `json:"name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Duration  int    `json:"duration"`
	Preview   *Link  `json:"preview"`
}

// Collection коллекция объектов
type Collection struct {
	ID         string        `json:"id"`
	Type       string        `json:"type"`
	TotalItems int           `json:"totalItems"`
	Items      []interface{} `json:"items,omitempty"`
	First      string        `json:"first,omitempty"`
	Next       string        `json:"next,omitempty"`
	Prev       string        `json:"prev,omitempty"`
}

// PublicKey публичный ключ
type PublicKey struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPem string `json:"publicKeyPem"`
}

// Endpoints конечные точки
type Endpoints struct {
	ProxyURL           string `json:"proxyUrl"`
	OauthAuthorization string `json:"oauthAuthorization"`
	OauthToken         string `json:"oauthToken"`
	ProvideClientKey   string `json:"provideClientKey"`
	SignClientKey      string `json:"signClientKey"`
	SharedInbox        string `json:"sharedInbox"`
}

// Tag тег (Hashtag, Mention и т.д.)
type Tag struct {
	Type    string     `json:"type"`
	Href    string     `json:"href"`
	Name    string     `json:"name"`
	Updated *time.Time `json:"updated"`
	Icon    *Link      `json:"icon"`
}

// Response ответ ActivityPub
type Response struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Actor     string      `json:"actor"`
	Object    interface{} `json:"object"`
	To        []string    `json:"to"`
	CC        []string    `json:"cc"`
	Published *time.Time  `json:"published"`
}

// === POOL'Ы ===

var contextPool = sync.Pool{
	New: func() interface{} {
		return &Context{
			response: &Response{},
		}
	},
}

// === НОВЫЙ КОНТЕКСТ ===

// NewContext создаёт новый ActivityPub контекст
func NewContext(ws *azurewebsockets.Conn, opcode int, data []byte) *Context {
	ctx := contextPool.Get().(*Context)
	ctx.ws = ws
	ctx.opcode = opcode
	ctx.data = data
	ctx.object = nil
	ctx.actor = nil
	ctx.response = &Response{}
	return ctx
}

// PutContext возвращает контекст в пул
func PutContext(ctx *Context) {
	ctx.ws = nil
	ctx.data = nil
	ctx.object = nil
	ctx.actor = nil
	ctx.response = &Response{}
	contextPool.Put(ctx)
}

// === МЕТОДЫ CONTEXT ===

// Bind парсит JSON-LD сообщение в структуру
func (c *Context) Bind(v interface{}) error {
	return json.Unmarshal(c.data, v)
}

// BindObject парсит ActivityPub объект
func (c *Context) BindObject() (*Object, error) {
	if c.object != nil {
		return c.object, nil
	}

	var obj Object
	if err := json.Unmarshal(c.data, &obj); err != nil {
		return nil, err
	}

	c.object = &obj
	return &obj, nil
}

// BindActor парсит ActivityPub актёра
func (c *Context) BindActor() (*Actor, error) {
	if c.actor != nil {
		return c.actor, nil
	}

	var actor Actor
	if err := json.Unmarshal(c.data, &actor); err != nil {
		return nil, err
	}

	c.actor = &actor
	return &actor, nil
}

// GetOpcode возвращает тип WebSocket сообщения
func (c *Context) GetOpcode() int {
	return c.opcode
}

// GetData возвращает сырые данные
func (c *Context) GetData() []byte {
	return c.data
}

// GetWebSocket возвращает WebSocket соединение
func (c *Context) GetWebSocket() *azurewebsockets.Conn {
	return c.ws
}

// SetAzureContext устанавливает Azure контекст
func (c *Context) SetAzureContext(ctx *azure.Context) {
	c.azureCtx = ctx
}

// GetAzureContext возвращает Azure контекст
func (c *Context) GetAzureContext() *azure.Context {
	return c.azureCtx
}

// Outbox отправляет сообщение в outbox (ответ клиенту)
func (c *Context) Outbox(v interface{}) error {
	// Сохраняем ответ для HTTP
	c.httpResp = v

	// Если это WebSocket - отправляем сразу
	if c.ws != nil {
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return c.ws.WriteMessage(azurewebsockets.OpText, data)
	}

	return nil
}

// Send отправляет произвольное сообщение
func (c *Context) Send(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.ws.WriteMessage(azurewebsockets.OpText, data)
}

// SendText отправляет текстовое сообщение
func (c *Context) SendText(text string) error {
	return c.ws.WriteText([]byte(text))
}

// SendBinary отправляет бинарные данные
func (c *Context) SendBinary(data []byte) error {
	return c.ws.WriteBinary(data)
}

// Close закрывает WebSocket соединение
func (c *Context) Close() error {
	return c.ws.Close()
}

// === ACTIVITYPUB CONFIG ===

// Config конфигурация ActivityPub
type Config struct {
	// Domain домен сервера
	Domain string

	// ActorPath путь к актёру (по умолчанию /actor)
	ActorPath string

	// InboxPath путь к inbox (по умолчанию /inbox)
	InboxPath string

	// OutboxPath путь к outbox (по умолчанию /outbox)
	OutboxPath string

	// FollowingPath путь к following
	FollowingPath string

	// FollowersPath путь к followers
	FollowersPath string

	// EnableSharedInbox включить shared inbox
	EnableSharedInbox bool

	// SharedInboxPath путь к shared inbox
	SharedInboxPath string

	// MaxMessageSize максимальный размер сообщения
	MaxMessageSize int

	// Timeout таймаут операций
	Timeout time.Duration

	// AutoAcceptFollow автоматически принимать follow запросы
	AutoAcceptFollow bool

	// OnError обработчик ошибок
	OnError func(error)
}

// DefaultConfig конфигурация по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Domain:            "localhost:7070",
		ActorPath:         "/actor",
		InboxPath:         "/inbox",
		OutboxPath:        "/outbox",
		FollowingPath:     "/following",
		FollowersPath:     "/followers",
		EnableSharedInbox: true,
		SharedInboxPath:   "/shared/inbox",
		MaxMessageSize:    65536, // 64KB
		Timeout:           30 * time.Second,
		AutoAcceptFollow:  false,
		OnError: func(err error) {
			// Логирование ошибки по умолчанию
		},
	}
}

// === ACTIVITYPUB ===

// ActivityPub основной класс
type ActivityPub struct {
	config           *Config
	inboxHandler     func(*Context)
	outboxHandler    func(*Context)
	followingHandler func(*Context)
	followersHandler func(*Context)
	actorHandler     func(*Context)
	mu               sync.RWMutex
}

// New создаёт новый ActivityPub экземпляр
func New(config ...*Config) *ActivityPub {
	cfg := DefaultConfig()
	if len(config) > 0 && config[0] != nil {
		cfg = config[0]
	}

	return &ActivityPub{
		config: cfg,
	}
}

// Inbox устанавливает обработчик входящих сообщений и возвращает ap для цепочки
func (ap *ActivityPub) Inbox(handler func(*Context)) *ActivityPub {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.inboxHandler = handler
	return ap
}

// Outbox устанавливает обработчик исходящих сообщений и возвращает ap для цепочки
func (ap *ActivityPub) Outbox(handler func(*Context)) *ActivityPub {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.outboxHandler = handler
	return ap
}

// Following устанавливает обработчик following и возвращает ap для цепочки
func (ap *ActivityPub) Following(handler func(*Context)) *ActivityPub {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.followingHandler = handler
	return ap
}

// Follows устанавливает обработчик followers и возвращает ap для цепочки (алиас для Followers)
func (ap *ActivityPub) Follows(handler func(*Context)) *ActivityPub {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.followersHandler = handler
	return ap
}

// Actor устанавливает обработчик actor и возвращает ap для цепочки
func (ap *ActivityPub) Actor(handler func(*Context)) *ActivityPub {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.actorHandler = handler
	return ap
}

// Followers устанавливает обработчик followers и возвращает ap для цепочки
func (ap *ActivityPub) Followers(handler func(*Context)) *ActivityPub {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.followersHandler = handler
	return ap
}

// GetConfig возвращает конфигурацию
func (ap *ActivityPub) GetConfig() *Config {
	return ap.config
}

// === MIDDLEWARE ===

// Middleware создаёт Azure middleware для ActivityPub (поддерживает HTTP + WebSocket)
func Middleware(ap *ActivityPub) azure.Middleware {
	return func(c *azure.Context, next ultrahttp.RouteHandler) {
		path := c.GetUltra().GetPath()
		println("DEBUG Middleware: path =", path, "ActorPath =", ap.config.ActorPath)

		// Проверяем путь
		if path == ap.config.InboxPath || path == ap.config.SharedInboxPath {
			println("DEBUG: Matching inbox")
			// Inbox handler (поддерживает HTTP и WebSocket)
			handleInbox(c, ap)
			return
		}

		if path == ap.config.OutboxPath {
			println("DEBUG: Matching outbox")
			// Outbox handler
			handleOutbox(c, ap)
			return
		}

		if path == ap.config.FollowingPath {
			println("DEBUG: Matching following")
			// Following handler
			handleFollowing(c, ap)
			return
		}

		if path == ap.config.FollowersPath {
			println("DEBUG: Matching followers")
			// Followers handler
			handleFollowers(c, ap)
			return
		}

		if path == ap.config.ActorPath {
			println("DEBUG: Matching actor")
			// Actor handler
			handleActor(c, ap)
			return
		}

		println("DEBUG: No match, calling next")
		// Не ActivityPub маршрут
		next(c.GetUltra())
	}
}

// MiddlewarePath создаёт middleware для конкретного пути
func MiddlewarePath(path string, ap *ActivityPub) azure.Middleware {
	return func(c *azure.Context, next ultrahttp.RouteHandler) {
		if c.GetUltra().GetPath() == path {
			handleInbox(c, ap)
			return
		}
		next(c.GetUltra())
	}
}

// === HANDLERS ===

// handleInbox обрабатывает inbox запросы (HTTP POST + WebSocket)
func handleInbox(c *azure.Context, ap *ActivityPub) {
	// Проверяем WebSocket upgrade
	upgrade := c.GetHeader("Upgrade")

	if upgrade != "" {
		// WebSocket подключение
		handleInboxWebSocket(c, ap)
		return
	}

	// HTTP POST запрос (классический ActivityPub)
	handleInboxHTTP(c, ap)
}

// handleInboxWebSocket обрабатывает WebSocket подключения
func handleInboxWebSocket(c *azure.Context, ap *ActivityPub) {
	// Получаем WebSocket соединение
	ws, err := azurewebsockets.Upgrade(c.GetUltra())
	if err != nil {
		c.SetStatus(400, "Bad Request")
		c.Json(azure.M{
			"error": err.Error(),
		})
		return
	}

	// Создаём ActivityPub контекст
	ctx := NewContext(ws, azurewebsockets.OpText, nil)
	ctx.SetAzureContext(c)
	defer PutContext(ctx)

	// Вызываем inbox handler в горутине
	go func() {
		for {
			opcode, data, err := ws.ReadMessage()
			if err != nil {
				ws.Close()
				return
			}

			ctx.opcode = opcode
			ctx.data = data

			ap.mu.RLock()
			handler := ap.inboxHandler
			ap.mu.RUnlock()

			if handler != nil {
				handler(ctx)
			}
		}
	}()

	// Для WebSocket - блокируем HTTP ответ
	// Соединение остаётся открытым
	select {}
}

// handleInboxHTTP обрабатывает HTTP POST запросы
func handleInboxHTTP(c *azure.Context, ap *ActivityPub) {
	// Проверяем метод
	method := c.GetUltra().GetMethod()
	if method != "POST" {
		c.SetStatus(405, "Method Not Allowed")
		c.Json(azure.M{
			"error": "Method not allowed. Use POST for ActivityPub inbox.",
		})
		return
	}

	// Получаем тело запроса
	body := c.GetBody()
	if len(body) == 0 {
		c.SetStatus(400, "Bad Request")
		c.Json(azure.M{
			"error": "Empty request body",
		})
		return
	}

	// Создаём контекст
	ctx := contextPool.Get().(*Context)
	ctx.SetAzureContext(c)
	ctx.data = body
	defer PutContext(ctx)

	// Вызываем handler
	ap.mu.RLock()
	handler := ap.inboxHandler
	ap.mu.RUnlock()

	if handler != nil {
		handler(ctx)
	}

	// Отправляем ответ (если handler не отправил свой)
	if ctx.response != nil && ctx.response.Type != "" {
		c.JsonStatus(200, azure.M{
			"id":     ctx.response.ID,
			"type":   ctx.response.Type,
			"actor":  ctx.response.Actor,
			"object": ctx.response.Object,
		})
	} else {
		c.JsonStatus(200, azure.M{
			"status": "accepted",
		})
	}
}

// handleOutbox обрабатывает outbox запросы
func handleOutbox(c *azure.Context, ap *ActivityPub) {
	ap.mu.RLock()
	handler := ap.outboxHandler
	ap.mu.RUnlock()

	if handler == nil {
		c.SetStatus(404, "Not Found")
		c.Json(azure.M{
			"error": "Outbox handler not configured",
		})
		return
	}

	// Создаём контекст
	ctx := contextPool.Get().(*Context)
	ctx.SetAzureContext(c)
	defer PutContext(ctx)

	// Вызываем handler
	handler(ctx)

	// Отправляем HTTP ответ
	if ctx.httpResp != nil {
		if resp, ok := ctx.httpResp.(azure.M); ok {
			c.JsonStatus(200, resp)
		} else {
			c.JsonStatus(200, azure.M{"data": ctx.httpResp})
		}
	}
}

// handleFollowing обрабатывает following запросы
func handleFollowing(c *azure.Context, ap *ActivityPub) {
	ap.mu.RLock()
	handler := ap.followingHandler
	ap.mu.RUnlock()

	if handler == nil {
		c.SetStatus(404, "Not Found")
		c.Json(azure.M{
			"error": "Following handler not configured",
		})
		return
	}

	ctx := contextPool.Get().(*Context)
	ctx.SetAzureContext(c)
	defer PutContext(ctx)

	handler(ctx)

	// Отправляем HTTP ответ
	if ctx.httpResp != nil {
		if resp, ok := ctx.httpResp.(azure.M); ok {
			c.JsonStatus(200, resp)
		} else {
			c.JsonStatus(200, azure.M{"data": ctx.httpResp})
		}
	}
}

// handleFollowers обрабатывает followers запросы
func handleFollowers(c *azure.Context, ap *ActivityPub) {
	ap.mu.RLock()
	handler := ap.followersHandler
	ap.mu.RUnlock()

	if handler == nil {
		c.SetStatus(404, "Not Found")
		c.Json(azure.M{
			"error": "Followers handler not configured",
		})
		return
	}

	ctx := contextPool.Get().(*Context)
	ctx.SetAzureContext(c)
	defer PutContext(ctx)

	handler(ctx)

	// Отправляем HTTP ответ
	if ctx.httpResp != nil {
		if resp, ok := ctx.httpResp.(azure.M); ok {
			c.JsonStatus(200, resp)
		} else {
			c.JsonStatus(200, azure.M{"data": ctx.httpResp})
		}
	}
}

// handleActor обрабатывает actor запросы
func handleActor(c *azure.Context, ap *ActivityPub) {
	ap.mu.RLock()
	handler := ap.actorHandler
	ap.mu.RUnlock()

	if handler == nil {
		c.SetStatus(404, "Not Found")
		c.Json(azure.M{
			"error": "Actor handler not configured",
		})
		return
	}

	ctx := contextPool.Get().(*Context)
	ctx.SetAzureContext(c)
	defer PutContext(ctx)

	// Вызываем handler
	println("DEBUG: Calling actor handler")
	handler(ctx)
	println("DEBUG: handler returned, httpResp type =", fmt.Sprintf("%T", ctx.httpResp))

	// Отправляем HTTP ответ - устанавливаем напрямую в ultrahttp
	if ctx.httpResp != nil {
		println("DEBUG: Sending response")
		if resp, ok := ctx.httpResp.(azure.M); ok {
			c.JsonStatus(200, resp)
			println("DEBUG: Response sent")
		} else {
			c.JsonStatus(200, azure.M{"error": "Invalid response type"})
		}
	} else {
		println("DEBUG: No httpResp!")
		c.JsonStatus(200, azure.M{"error": "No response"})
	}
}

// === ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ ===

// NewObject создаёт новый ActivityPub объект
func NewObject(objType string) *Object {
	return &Object{
		Type:  objType,
		To:    make([]string, 0),
		CC:    make([]string, 0),
		Tag:   make([]Tag, 0),
		Extra: make(map[string]interface{}),
	}
}

// NewActivity создаёт новую Activity
func NewActivity(activityType string, actor string, object interface{}) *Object {
	return &Object{
		Type:      activityType,
		Actor:     actor,
		Object:    object,
		To:        make([]string, 0),
		CC:        make([]string, 0),
		Published: ptrTime(time.Now()),
	}
}

// NewActor создаёт нового актёра
func NewActor(actorType, id, name, username string) *Actor {
	return &Actor{
		ID:                id,
		Type:              actorType,
		Name:              name,
		PreferredUsername: username,
		Inbox:             id + "/inbox",
		Outbox:            id + "/outbox",
		Following:         id + "/following",
		Followers:         id + "/followers",
		To:                make([]string, 0),
		CC:                make([]string, 0),
	}
}

// ptrTime возвращает указатель на время
func ptrTime(t time.Time) *time.Time {
	return &t
}

// MarshalJSON-LD сериализует объект в JSON-LD
func MarshalJSONLD(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

// UnmarshalJSON-LD парсит JSON-LD объект
func UnmarshalJSONLD(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// === МЕТОДЫ ДЛЯ WORKERS/FOLLOWERS ===

// SendToFollowers отправляет сообщение всем подписчикам
func (c *Context) SendToFollowers(activity *Object) error {
	// Создаём коллекцию для отправки
	collection := &Collection{
		Type:       "OrderedCollection",
		TotalItems: 1,
		Items:      []interface{}{activity},
	}

	return c.Outbox(collection)
}

// AddFollower добавляет подписчика (для использования в обработчиках)
func (c *Context) AddFollower(actorID string) error {
	// Здесь должна быть логика сохранения в БД
	// Для теперь просто логируем
	return nil
}

// RemoveFollower удаляет подписчика
func (c *Context) RemoveFollower(actorID string) error {
	return nil
}

// GetFollowers возвращает список подписчиков
func (c *Context) GetFollowers() ([]string, error) {
	// Здесь должна быть логика получения из БД
	return []string{}, nil
}

// GetFollowing возвращает список подписок
func (c *Context) GetFollowing() ([]string, error) {
	// Здесь должна быть логика получения из БД
	return []string{}, nil
}

// Follow подписывается на актёра
func (c *Context) Follow(actorID string) error {
	follow := NewActivity(ActivityFollow, c.azureCtx.GetHeader("Host"), actorID)
	return c.Outbox(follow)
}

// Unfollow отписывается от актёра
func (c *Context) Unfollow(actorID string) error {
	// Создаём Undo для Follow
	undo := NewActivity(ActivityUndo, c.azureCtx.GetHeader("Host"), M{
		"type":   ActivityFollow,
		"actor":  c.azureCtx.GetHeader("Host"),
		"object": actorID,
	})
	return c.Outbox(undo)
}

// AcceptFollow принимает запрос на подписку
func (c *Context) AcceptFollow(followActivity *Object) error {
	accept := NewActivity(ActivityAccept, followActivity.Object.(string), followActivity)
	return c.Outbox(accept)
}

// RejectFollow отклоняет запрос на подписку
func (c *Context) RejectFollow(followActivity *Object) error {
	reject := NewActivity(ActivityReject, followActivity.Object.(string), followActivity)
	return c.Outbox(reject)
}
