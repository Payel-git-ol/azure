package aurum

import (
	"time"
)

// Пример использования Aurum ORM
func ExampleUsage() {
	// Подключение к базе данных
	var db DatabaseConnection

	// Вариант 1: Подключение по DNS строке
	// env.Load()
	// dns := env.MustGet("DNS", "")
	// _ = db.Connection(dns)
	_ = db

	// Вариант 2: Декларативное подключение
	_ = db.ConnectionDeclarative(Connection{
		Host:     "localhost",
		Port:     "5432",
		Name:     "postgres",
		Password: "password",
		Database: "mydb",
		Driver:   "postgres",
	})

	// Автоматическая миграция моделей
	_ = db.AutoMigrate(&User{}, &Post{})

	// Получение пользователя по ID
	GetUserById := func(id int) (*User, error) {
		aurum := Model[User](db.GetDB())
		return aurum.GetById(id)
	}

	// Создание пользователя
	UserCreate := func(name string, age int, password string) error {
		aurum := Model[User](db.GetDB())
		_, err := aurum.
			ParamCreate("name", name).
			ParamCreate("age", age).
			ParamCreate("password", password).
			Create()
		return err
	}

	// Обновление пользователя
	UpdateUser := func(userName string, newAge int) error {
		aurum := Model[User](db.GetDB())
		user, err := aurum.GetData("name", userName)
		if err != nil {
			return err
		}
		return aurum.UpdateData(user, map[string]any{
			"age": newAge,
		})
	}

	// Получение всех пользователей
	GetAllUsers := func() ([]*User, error) {
		aurum := Model[User](db.GetDB())
		return aurum.GetAll()
	}

	// Поиск с условиями
	FindUsers := func() ([]*User, error) {
		aurum := Model[User](db.GetDB())
		return aurum.
			Where("age > ?", 18).
			OrderBy("name ASC").
			Limit(10).
			Offset(0).
			GetAll()
	}

	// Транзакция
	TransactionalOperation := func() error {
		aurum := Model[User](db.GetDB())
		return aurum.Transaction(func(tx *Aurum[User]) error {
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

	// Хуки
	_ = func() {
		aurum := Model[User](db.GetDB())
		aurum.
			BeforeCreate(func(user *User) {
				// Логика перед созданием
				user.CreatedAt = nil
			}).
			AfterUpdate(func(user *User) {
				// Логика после обновления
			})
	}

	// Удаление
	DeleteUser := func(id int) error {
		aurum := Model[User](db.GetDB())
		return aurum.DeleteById(id)
	}

	// Подсчёт записей
	CountUsers := func() (int64, error) {
		aurum := Model[User](db.GetDB())
		return aurum.Where("age > ?", 18).Count()
	}

	// Проверка существования
	UserExists := func(id int) (bool, error) {
		aurum := Model[User](db.GetDB())
		return aurum.Exists(id)
	}

	// Первая запись
	FirstUser := func() (*User, error) {
		aurum := Model[User](db.GetDB())
		return aurum.First()
	}

	// Поиск по условиям
	Find := func() ([]*User, error) {
		aurum := Model[User](db.GetDB())
		return aurum.Find(map[string]any{
			"age":  25,
			"name": "John",
		})
	}

	// Мягкое удаление (soft delete)
	SoftDeleteUser := func(id int) error {
		aurum := Model[User](db.GetDB())
		return aurum.SoftDelete(id)
	}

	// Получение с удалёнными записями
	WithDeletedUsers := func() ([]*User, error) {
		aurum := Model[User](db.GetDB())
		return aurum.WithDeleted().GetAll()
	}

	// Восстановление после soft delete
	RestoreUser := func(id int) error {
		aurum := Model[User](db.GetDB())
		return aurum.Restore(id)
	}

	// Агрегатные функции
	SumAge := func() (float64, error) {
		aurum := Model[User](db.GetDB())
		return aurum.Sum("age")
	}

	AvgAge := func() (float64, error) {
		aurum := Model[User](db.GetDB())
		return aurum.Avg("age")
	}

	// Group By
	GroupByAge := func() ([]*User, error) {
		aurum := Model[User](db.GetDB())
		return aurum.GroupBy("age")
	}

	// Bulk операции
	BulkCreate := func(users []*User) error {
		aurum := Model[User](db.GetDB())
		return aurum.BulkCreate(users)
	}

	BulkUpdate := func(users []*User) error {
		aurum := Model[User](db.GetDB())
		return aurum.BulkUpdate(users)
	}

	BulkDelete := func(ids []int) error {
		aurum := Model[User](db.GetDB())
		return aurum.BulkDelete(ids)
	}

	// Preload связанных данных
	WithPosts := func() ([]*User, error) {
		aurum := Model[User](db.GetDB())
		return aurum.Preload("Posts").GetAll()
	}

	// Joins
	WithJoins := func() ([]*User, error) {
		aurum := Model[User](db.GetDB())
		return aurum.Joins("posts", "comments").GetAll()
	}

	// Raw SQL
	RawQuery := func() ([]*User, error) {
		aurum := Model[User](db.GetDB())
		return aurum.Query("SELECT * FROM users WHERE age > $1", 18)
	}

	_ = GetUserById
	_ = UserCreate
	_ = UpdateUser
	_ = GetAllUsers
	_ = FindUsers
	_ = TransactionalOperation
	_ = DeleteUser
	_ = CountUsers
	_ = UserExists
	_ = FirstUser
	_ = Find
	_ = SoftDeleteUser
	_ = WithDeletedUsers
	_ = RestoreUser
	_ = SumAge
	_ = AvgAge
	_ = GroupByAge
	_ = BulkCreate
	_ = BulkUpdate
	_ = BulkDelete
	_ = WithPosts
	_ = WithJoins
	_ = RawQuery
}

// Пример модели User
type User struct {
	ID        int       `aurum:"id"`
	Name      string    `aurum:"name"`
	Age       int       `aurum:"age"`
	Password  string    `aurum:"password"`
	Email     string    `aurum:"email"`
	CreatedAt *time.Time `aurum:"created_at"`
	UpdatedAt *time.Time `aurum:"updated_at"`
	DeletedAt *time.Time `aurum:"deleted_at"`
	Posts     []*Post   `aurum:"-"` // Связь
}

// Пример модели Post
type Post struct {
	ID        int        `aurum:"id"`
	Title     string     `aurum:"title"`
	Content   string     `aurum:"content"`
	UserID    int        `aurum:"user_id"`
	CreatedAt *time.Time `aurum:"created_at"`
	UpdatedAt *time.Time `aurum:"updated_at"`
	DeletedAt *time.Time `aurum:"deleted_at"`
}
