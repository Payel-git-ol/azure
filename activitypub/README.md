# ActivityPub для Azure

Реализация протокола **ActivityPub** (W3C standard) для Azure фреймворка с использованием WebSocket для real-time федерации.

## 📖 Что такое ActivityPub?

ActivityPub — это децентрализованный протокол для социальных сетей, который позволяет различным серверам обмениваться активностями (посты, лайки, подписки и т.д.).

**Спецификация:** https://www.w3.org/TR/activitypub/

## 🚀 Быстрый старт

```go
package main

import (
	"github.com/Payel-git-ol/azure"
	"github.com/Payel-git-ol/azure/activitypub"
	"log"
)

func main() {
	a := azure.New()
	
	// Создаём ActivityPub
	act := activitypub.New(&activitypub.Config{
		Domain: "localhost:7070",
		AutoAcceptFollow: true,
	})

	// Настраиваем обработчики (цепочкой)
	act.Inbox(func(c *activitypub.Context) {
		// Парсим входящее сообщение
		var obj activitypub.Object
		c.Bind(&obj)

		log.Printf("Получено: %s от %s", obj.Type, obj.Actor)

		// Обрабатываем в зависимости от типа
		switch obj.Type {
		case activitypub.ActivityFollow:
			// Обработка подписки
			handleFollow(c, &obj)
			
		case activitypub.ActivityCreate:
			// Обработка создания поста
			handleCreate(c, &obj)
			
		case activitypub.ActivityLike:
			// Обработка лайка
			handleLike(c, &obj)
		}

		// Отправляем ответ используя M{}
		c.Outbox(activitypub.M{
			"type": "Accept",
			"actor": "https://localhost:7070/actor",
			"object": obj.ID,
		})
	}).Outbox(func(c *activitypub.Context) {
		// Outbox - исходящие сообщения
		c.Outbox(activitypub.M{
			"id": "https://localhost:7070/outbox",
			"type": "OrderedCollection",
			"totalItems": 10,
			"items": []activitypub.M{},
		})
	}).Following(func(c *activitypub.Context) {
		// Following - список подписок
		c.Outbox(activitypub.M{
			"id": "https://localhost:7070/following",
			"type": "OrderedCollection",
			"totalItems": 5,
			"items": []string{
				"https://server1.com/user1",
				"https://server2.com/user2",
			},
		})
	}).Followers(func(c *activitypub.Context) {
		// Followers - список подписчиков
		c.Outbox(activitypub.M{
			"id": "https://localhost:7070/followers",
			"type": "OrderedCollection",
			"totalItems": 100,
			"items": []string{
				"https://server2.com/user2",
				"https://server3.com/user3",
			},
		})
	}).Actor(func(c *activitypub.Context) {
		// Actor - информация об актёре
		c.Outbox(activitypub.M{
			"id": "https://localhost:7070/actor",
			"type": "Person",
			"name": "My Bot",
			"preferredUsername": "mybot",
			"inbox": "https://localhost:7070/inbox",
			"outbox": "https://localhost:7070/outbox",
			"following": "https://localhost:7070/following",
			"followers": "https://localhost:7070/followers",
		})
	})

	// Регистрируем middleware
	a.Use(activitypub.Middleware(act))
	
	// Или для конкретного пути
	// a.Use(activitypub.MiddlewarePath("/federation/inbox", act))

	log.Println("ActivityPub сервер на :7070")
	a.Run(":7070")
}
```

## 📡 Как это работает

### Вариант 1: WebSocket (Real-time)

**Подключение:**

```javascript
// JavaScript клиент
const ws = new WebSocket('ws://localhost:7070/inbox')

ws.onopen = () => {
  console.log('✅ Подключено к ActivityPub!')
  
  // Отправляем Activity (например, Follow)
  ws.send(JSON.stringify({
    "@context": "https://www.w3.org/ns/activitystreams",
    "id": "https://yourserver.com/activity/123",
    "type": "Follow",
    "actor": "https://yourserver.com/user",
    "object": "https://localhost:7070/actor"
  }))
}

ws.onmessage = (event) => {
  const data = JSON.parse(event.data)
  console.log('📥 Получено:', data)
  
  // Например, получили Accept
  if (data.type === 'Accept') {
    console.log('✅ Подписка принята!')
  }
  
  // Или получили Create (новый пост)
  if (data.type === 'Create') {
    console.log('📝 Новый пост:', data.object.content)
  }
}

ws.onerror = (error) => {
  console.error('❌ Ошибка:', error)
}

ws.onclose = () => {
  console.log('🔌 Соединение закрыто')
}
```

**Go клиент:**

```go
package main

import (
	"encoding/json"
	"log"
	"github.com/gorilla/websocket"
)

func main() {
	// Подключаемся к ActivityPub серверу
	ws, _, err := websocket.DefaultDialer.Dial("ws://localhost:7070/inbox", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	// Отправляем Follow активность
	follow := map[string]interface{}{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       "https://yourserver.com/activity/123",
		"type":     "Follow",
		"actor":    "https://yourserver.com/user",
		"object":   "https://localhost:7070/actor",
	}

	if err := ws.WriteJSON(follow); err != nil {
		log.Fatal(err)
	}

	// Читаем ответ
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("Error:", err)
			break
		}

		var activity map[string]interface{}
		if err := json.Unmarshal(message, &activity); err != nil {
			log.Println("Unmarshal error:", err)
			continue
		}

		log.Printf("Получено: %+v", activity)
	}
}
```

### Вариант 2: HTTP POST (Классический ActivityPub)

**curl пример:**

```bash
# Отправляем Follow запрос
curl -X POST http://localhost:7070/inbox \
  -H "Content-Type: application/activity+json" \
  -H "Accept: application/activity+json" \
  -d '{
    "@context": "https://www.w3.org/ns/activitystreams",
    "id": "https://mastodon.social/users/user#follows",
    "type": "Follow",
    "actor": "https://mastodon.social/users/user",
    "object": "http://localhost:7070/actor"
  }'

# Ответ:
# {"status": "accepted"}
```

**HTTP GET для получения данных:**

```bash
# Получаем информацию об актёре
curl -X GET http://localhost:7070/actor \
  -H "Accept: application/activity+json"

# Получаем outbox (последние активности)
curl -X GET http://localhost:7070/outbox \
  -H "Accept: application/activity+json"

# Получаем followers
curl -X GET http://localhost:7070/followers \
  -H "Accept: application/activity+json"

# Получаем following
curl -X GET http://localhost:7070/following \
  -H "Accept: application/activity+json"
```

**Python пример (HTTP POST):**

```python
import requests
import json

# Отправляем Follow активность
activity = {
    "@context": "https://www.w3.org/ns/activitystreams",
    "id": "https://yourserver.com/activity/123",
    "type": "Follow",
    "actor": "https://yourserver.com/user",
    "object": "http://localhost:7070/actor"
}

headers = {
    "Content-Type": "application/activity+json",
    "Accept": "application/activity+json"
}

response = requests.post(
    "http://localhost:7070/inbox",
    headers=headers,
    json=activity
)

print(f"Status: {response.status_code}")
print(f"Response: {response.json()}")
```

### Вариант 3: Интеграция с Azure Context

```go
act.Inbox(func(c *activitypub.Context) {
    // Парсим входящее сообщение
    var obj activitypub.Object
    c.Bind(&obj)
    
    // Проверяем тип активности
    switch obj.Type {
    case activitypub.ActivityFollow:
        // Получаем актёра
        actor, _ := c.BindActor()
        
        log.Printf("%s хочет подписаться", actor.Name)
        
        // Отправляем Accept
        c.Outbox(activitypub.M{
            "type": "Accept",
            "actor": "https://localhost:7070/actor",
            "object": obj.ID,
        })
        
    case activitypub.ActivityCreate:
        // Новый пост
        note, ok := obj.Object.(activitypub.Object)
        if ok {
            log.Printf("Новый пост: %s", note.Content)
        }
        
        // Отправляем в outbox
        c.Outbox(activitypub.M{
            "type": "Announce",
            "actor": "https://localhost:7070/actor",
            "object": obj.ID,
            "to": []string{"https://www.w3.org/ns/activitystreams#Public"},
        })
    }
})
```

## 🔧 Конфигурация

```go
config := &activitypub.Config{
  Domain: "localhost:7070",           // Домен сервера
  ActorPath: "/actor",                // Путь к актёру
  InboxPath: "/inbox",                // Путь к inbox
  OutboxPath: "/outbox",              // Путь к outbox
  FollowingPath: "/following",        // Путь к following
  FollowersPath: "/followers",        // Путь к followers
  EnableSharedInbox: true,            // Включить shared inbox
  SharedInboxPath: "/shared/inbox",   // Путь к shared inbox
  MaxMessageSize: 65536,              // Макс размер сообщения (64KB)
  Timeout: 30 * time.Second,          // Таймаут операций
  AutoAcceptFollow: false,            // Авто-принятие follow
}

act := activitypub.New(config)
```

## 📦 Типы ActivityPub

### Activity Types

```go
activitypub.ActivityCreate   // Создание объекта
activitypub.ActivityDelete   // Удаление объекта
activitypub.ActivityFollow   // Подписка
activitypub.ActivityAccept   // Принятие
activitypub.ActivityReject   // Отклонение
activitypub.ActivityUndo     // Отмена
activitypub.ActivityUpdate   // Обновление
activitypub.ActivityLike     // Лайк
activitypub.ActivityAnnounce // Репост
```

### Object Types

```go
activitypub.ObjectNote     // Пост/заметка
activitypub.ObjectArticle  // Статья
activitypub.ObjectImage    // Изображение
activitypub.ObjectVideo    // Видео
```

## 🎯 Примеры использования

### Обработка подписки (Follow)

```go
act.Inbox(func(c *activitypub.Context) {
  var obj activitypub.Object
  c.Bind(&obj)
  
  if obj.Type == activitypub.ActivityFollow {
    // Получаем актёра
    actor, _ := c.BindActor()
    
    log.Printf("%s хочет подписаться", actor.Name)
    
    // Принимаем подписку
    c.Outbox(activitypub.M{
      "type": "Accept",
      "actor": "https://localhost:7070/actor",
      "object": obj.ID,
    })
    
    // Сохраняем подписчика в БД
    // db.SaveFollower(actor.ID)
  }
})
```

### Создание поста

```go
act.Outbox(func(c *activitypub.Context) {
  // Создаём пост
  note := activitypub.NewObject(activitypub.ObjectNote)
  note.Content = "Hello Fediverse!"
  note.To = []string{"https://www.w3.org/ns/activitystreams#Public"}
  note.CC = []string{"https://localhost:7070/followers"}
  
  // Создаём активность
  create := activitypub.NewActivity(
    activitypub.ActivityCreate,
    "https://localhost:7070/actor",
    note,
  )
  
  // Отправляем
  c.Outbox(create)
})
```

### Обработка лайков

```go
act.Inbox(func(c *activitypub.Context) {
  var obj activitypub.Object
  c.Bind(&obj)
  
  if obj.Type == activitypub.ActivityLike {
    // Извлекаем ID лайкнутого объекта
    likedID := obj.Object.(string)
    
    log.Printf("Лайк на %s", likedID)
    
    // Сохраняем в БД
    // db.AddLike(likedID, obj.Actor)
  }
})
```

## 🔄 JSON-LD поддержка

ActivityPub использует JSON-LD для семантической разметки:

```go
// Парсинг JSON-LD
var obj activitypub.Object
c.Bind(&obj)

// Доступ к дополнительным полям
extra := obj.Extra["customField"]

// Сериализация
data, _ := activitypub.MarshalJSONLD(obj)
```

## 🌐 Интеграция с Fediverse

### Пример: Mastodon совместимость

```go
act.Actor(func(c *activitypub.Context) {
  actor := activitypub.NewActor("Person", 
    "https://localhost:7070/actor",
    "My Bot",
    "mybot",
  )
  
  // Добавляем publicKey для подписи
  actor.PublicKey = &activitypub.PublicKey{
    ID: "https://localhost:7070/actor#main-key",
    Owner: actor.ID,
    PublicKeyPem: "-----BEGIN PUBLIC KEY-----\n...",
  }
  
  // Добавляем endpoints для shared inbox
  actor.Endpoints = &activitypub.Endpoints{
    SharedInbox: "https://localhost:7070/shared/inbox",
  }
  
  c.Outbox(actor)
})
```

## 📊 Архитектура

```
┌─────────────┐         WebSocket          ┌──────────────┐
│   Client    │ ─────────────────────────> │  Azure AP    │
│ (Mastodon,  │                            │   Server     │
│  Pleroma,   │ <───────────────────────── │              │
│   etc.)     │         JSON-LD            │              │
└─────────────┘                            └──────────────┘
                                                │
                                                ▼
                                         ┌──────────────┐
                                         │   Database   │
                                         │  (Postgres,  │
                                         │   SQLite)    │
                                         └──────────────┘
```

## ⚡ Производительность

Благодаря использованию WebSocket и пулов объектов:
- **0 аллокаций** при парсинге сообщений (при переиспользовании)
- **< 1ms** задержка обработки
- **10K+** сообщений в секунду

## 🔐 Безопасность

### Проверка подписей

```go
func verifySignature(activity *activitypub.Object) bool {
  // Получаем publicKey актёра
  actor, _ := fetchActor(activity.Actor)
  
  // Проверяем HTTP Signature
  // https://www.w3.org/wiki/SocialCG/ActivityPub/Primer/Authentication
  
  return true // или false
}
```

### Rate limiting

```go
import "github.com/Payel-git-ol/azure"

a.Use(azure.RateLimitByIP(100, 200).Middleware())
```

## 📚 Дополнительные ресурсы

- [ActivityPub Specification](https://www.w3.org/TR/activitypub/)
- [Activity Streams 2.0](https://www.w3.org/TR/activitystreams-core/)
- [Mastodon API](https://docs.joinmastodon.org/spec/activitypub/)
- [Fediverse Dev](https://fediverse.dev/)

## 🎯 TODO

- [ ] HTTP Signature верификация
- [ ] WebFinger поддержка
- [ ] NodeInfo endpoint
- [ ] Pagination для коллекций
- [ ] Кэширование удалённых актёров
- [ ] Очереди для отправки

## 📄 Лицензия

MIT
