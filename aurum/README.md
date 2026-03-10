# Aurum ORM

Мощная и быстрая ORM для Go с использованием дженериков. Часть экосистемы Azure Framework.

## Особенности

- ✅ **Дженерики** — типобезопасность без приведения типов
- ✅ **Chainable API** — удобный синтаксис в стиле fluent
- ✅ **AutoMigrate** — автоматическое создание таблиц
- ✅ **Hooks** — BeforeCreate, AfterUpdate, BeforeDelete, AfterDelete
- ✅ **Soft Delete** — мягкое удаление с возможностью восстановления
- ✅ **Transactions** — поддержка транзакций
- ✅ **Bulk операции** — массовое создание, обновление, удаление
- ✅ **Агрегатные функции** — Sum, Avg, Count
- ✅ **Group By** — группировка записей
- ✅ **Preload & Joins** — загрузка связанных данных
- ✅ **PostgreSQL & SQLite** — поддержка популярных БД

## Установка

```bash
go get github.com/Payel-git-ol/azure/aurum
```

## Быстрый старт

### Подключение к базе данных

```go
package main

import (
    "github.com/Payel-git-ol/azure/aurum"
    "github.com/Payel-git-ol/azure/env"
)

func InitDb() {
    env.Load()
    dns := env.MustGet("DNS", "")

    var db aurum.DatabaseConnection
    
    // Вариант 1: Подключение по DNS строке
    err := db.Connection(dns)
    
    // Вариант 2: Декларативное подключение
    err = db.ConnectionDeclarative(aurum.Connection{
        Host:     "localhost",
        Port:     "5432",
        Name:     "postgres",
        Password: "password",
        Database: "mydb",
        Driver:   "postgres",
    })
    
    // Автоматическая миграция моделей
    err = db.AutoMigrate(&User{}, &Post{})
}
```

### Модели

```go
type User struct {
    ID        int        `aurum:"id"`
    Name      string     `aurum:"name"`
    Age       int        `aurum:"age"`
    Password  string     `aurum:"password"`
    Email     string     `aurum:"email"`
    CreatedAt *time.Time `aurum:"created_at"`
    UpdatedAt *time.Time `aurum:"updated_at"`
    DeletedAt *time.Time `aurum:"deleted_at"`
    Posts     []*Post    `aurum:"-"` // Связь
}

type Post struct {
    ID        int        `aurum:"id"`
    Title     string     `aurum:"title"`
    Content   string     `aurum:"content"`
    UserID    int        `aurum:"user_id"`
    CreatedAt *time.Time `aurum:"created_at"`
    UpdatedAt *time.Time `aurum:"updated_at"`
    DeletedAt *time.Time `aurum:"deleted_at"`
}
```

### Создание записи

```go
func CreateUser(name string, age int, password string) error {
    db := aurum.Model[User](db.GetDB())
    _, err := db.
        ParamCreate("name", name).
        ParamCreate("age", age).
        ParamCreate("password", password).
        Create()
    return err
}
```

### Получение записи по ID

```go
func GetUserById(id int) (*User, error) {
    db := aurum.Model[User](db.GetDB())
    return db.GetById(id)
}
```

### Получение записи по полю

```go
func GetUserByName(name string) (*User, error) {
    db := aurum.Model[User](db.GetDB())
    return db.GetData("name", name)
}
```

### Обновление записи

```go
func UpdateUserAge(userName string, newAge int) error {
    db := aurum.Model[User](db.GetDB())
    user, err := db.GetData("name", userName)
    if err != nil {
        return err
    }
    return db.UpdateData(user, map[string]any{
        "age": newAge,
    })
}
```

### Удаление записи

```go
func DeleteUser(id int) error {
    db := aurum.Model[User](db.GetDB())
    return db.DeleteById(id)
}
```

## Продвинутое использование

### Условия, сортировка, лимиты

```go
func GetAdultUsers() ([]*User, error) {
    db := aurum.Model[User](db.GetDB())
    return db.
        Where("age > ?", 18).
        OrderBy("name ASC").
        Limit(10).
        Offset(0).
        GetAll()
}
```

### Транзакции

```go
func TransactionalOperation() error {
    db := aurum.Model[User](db.GetDB())
    return db.Transaction(func(tx *aurum.Aurum[User]) error {
        _, err := tx.
            ParamCreate("name", "Alice").
            ParamCreate("age", 25).
            ParamCreate("password", "secret").
            Create()
        if err != nil {
            return err
        }

        _, err = tx.
            ParamCreate("name", "Bob").
            ParamCreate("age", 30).
            ParamCreate("password", "secret2").
            Create()
        return err
    })
}
```

### Hooks

```go
func WithHooks() {
    db := aurum.Model[User](db.GetDB())
    db.
        BeforeCreate(func(user *User) {
            // Логика перед созданием
            fmt.Println("Creating user:", user.Name)
        }).
        AfterUpdate(func(user *User) {
            // Логика после обновления
            fmt.Println("Updated user:", user.Name)
        })
}
```

### Soft Delete

```go
// Мягкое удаление
func SoftDeleteUser(id int) error {
    db := aurum.Model[User](db.GetDB())
    return db.SoftDelete(id)
}

// Получение с удалёнными записями
func GetAllWithDeleted() ([]*User, error) {
    db := aurum.Model[User](db.GetDB())
    return db.WithDeleted().GetAll()
}

// Восстановление
func RestoreUser(id int) error {
    db := aurum.Model[User](db.GetDB())
    return db.Restore(id)
}
```

### Агрегатные функции

```go
func GetStats() {
    db := aurum.Model[User](db.GetDB())
    
    // Сумма
    sum, _ := db.Sum("age")
    
    // Среднее
    avg, _ := db.Avg("age")
    
    // Количество
    count, _ := db.Count()
    
    // Group By
    groups, _ := db.GroupBy("age")
}
```

### Bulk операции

```go
func BulkOperations() {
    db := aurum.Model[User](db.GetDB())
    
    // Массовое создание
    users := []*User{
        {Name: "Alice", Age: 25},
        {Name: "Bob", Age: 30},
    }
    db.BulkCreate(users)
    
    // Массовое обновление
    db.BulkUpdate(users)
    
    // Массовое удаление
    ids := []int{1, 2, 3}
    db.BulkDelete(ids)
}
```

### Preload и Joins

```go
func WithRelations() {
    db := aurum.Model[User](db.GetDB())
    
    // Preload связанных данных
    users, _ := db.Preload("Posts").GetAll()
    
    // Joins
    users, _ = db.Joins("posts", "comments").GetAll()
}
```

### Raw SQL

```go
func RawQuery() ([]*User, error) {
    db := aurum.Model[User](db.GetDB())
    return db.Query("SELECT * FROM users WHERE age > $1", 18)
}
```

## API Reference

### DatabaseConnection

| Метод | Описание |
|-------|----------|
| `Connection(dns string)` | Подключение по DNS строке |
| `ConnectionDeclarative(conn Connection)` | Декларативное подключение |
| `AutoMigrate(models ...any)` | Автоматическая миграция моделей |
| `GetDB() DBConnection` | Получение DBConnection |
| `Model[T any](db DBConnection) *Aurum[T]` | Создание экземпляра ORM для модели |

### Aurum[T]

| Метод | Описание |
|-------|----------|
| `GetById(id int)` | Получение записи по ID |
| `GetData(field, value)` | Получение записи по полю |
| `GetAll()` | Получение всех записей |
| `ParamCreate(field, value)` | Добавление поля для создания |
| `Create()` | Создание записи |
| `UpdateData(entity, data)` | Обновление записи |
| `DeleteById(id)` | Удаление по ID |
| `Where(condition, args)` | Добавление условия WHERE |
| `OrderBy(order)` | Сортировка |
| `Limit(n)` | Лимит записей |
| `Offset(n)` | Смещение |
| `Count()` | Количество записей |
| `Exists(id)` | Проверка существования |
| `First()` | Первая запись |
| `Find(conditions)` | Поиск по условиям |
| `Transaction(fn)` | Выполнение в транзакции |
| `SoftDelete(id)` | Мягкое удаление |
| `WithDeleted()` | Включить удалённые записи |
| `Restore(id)` | Восстановление записи |
| `Sum(field)` | Сумма по полю |
| `Avg(field)` | Среднее по полю |
| `GroupBy(fields)` | Группировка |
| `BulkCreate(entities)` | Массовое создание |
| `BulkUpdate(entities)` | Массовное обновление |
| `BulkDelete(ids)` | Массовое удаление |
| `Query(sql, args)` | Raw SQL запрос |
| `Joins(tables)` | Добавление JOIN |
| `Preload(field)` | Загрузка связанных данных |
| `BeforeCreate(fn)` | Хук перед созданием |
| `AfterUpdate(fn)` | Хук после обновления |

## Теги

| Тег | Описание |
|-----|----------|
| `aurum:"table_name"` | Имя таблицы |
| `aurum:"column_name"` | Имя колонки |
| `aurum:"-"` | Пропустить поле |

## Лицензия

MIT
