package aurum

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Connection содержит параметры подключения к БД
type Connection struct {
	Host     string
	Port     string
	Name     string
	Password string
	Database string
	Driver   string // "postgres", "sqlite", "mysql"
}

// DatabaseConnection представляет подключение к базе данных
type DatabaseConnection struct {
	db *DBWrapper
}

// Connection подключается к БД по DNS строке
func (dc *DatabaseConnection) Connection(dns string) error {
	db, err := sql.Open("postgres", dns)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	dc.db = &DBWrapper{db: db}
	return nil
}

// ConnectionDeclarative подключается к БД через структуру Connection
func (dc *DatabaseConnection) ConnectionDeclarative(conn Connection) error {
	var dns string
	var driverName string

	switch conn.Driver {
	case "postgres":
		driverName = "postgres"
		dns = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			conn.Host, conn.Port, conn.Name, conn.Password, conn.Database)
	case "sqlite":
		driverName = "sqlite"
		dns = conn.Database
	default:
		return fmt.Errorf("unsupported driver: %s", conn.Driver)
	}

	db, err := sql.Open(driverName, dns)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	dc.db = &DBWrapper{db: db}
	return nil
}

// AutoMigrate автоматически создаёт таблицы для моделей
func (dc *DatabaseConnection) AutoMigrate(models ...any) error {
	for _, model := range models {
		if err := migrateModel(dc.db.db, model); err != nil {
			return err
		}
	}
	return nil
}

// Aurum основного ORM интерфейс с дженериками
type Aurum[T any] struct {
	db         DBConnection
	tableName  string
	ctx        context.Context
	conditions []string
	args       []any
	orderBy    string
	limit      int
	offset     int
	preloads   []string
	hooks      hooks
	createData map[string]any
	updateData map[string]any
	entity     *T
	err        error
}

// DBConnection интерфейс для работы с БД
type DBConnection interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// TxWrapper обёртка над транзакцией
type TxWrapper struct {
	tx *sql.Tx
}

func (t *TxWrapper) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *TxWrapper) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

func (t *TxWrapper) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

func (t *TxWrapper) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return t.tx, nil
}

// DBWrapper обёртка над sql.DB
type DBWrapper struct {
	db *sql.DB
}

func (d *DBWrapper) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

func (d *DBWrapper) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

func (d *DBWrapper) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}

func (d *DBWrapper) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return d.db.BeginTx(ctx, opts)
}

type hooks struct {
	beforeCreate []func(any)
	afterCreate  []func(any)
	beforeUpdate []func(any)
	afterUpdate  []func(any)
	beforeDelete []func(any)
	afterDelete  []func(any)
}

// New создаёт новый экземпляр Aurum
func New[T any](db DBConnection) *Aurum[T] {
	var entity T
	tableName := getTableName(entity)

	return &Aurum[T]{
		db:         db,
		tableName:  tableName,
		ctx:        context.Background(),
		conditions: make([]string, 0),
		args:       make([]any, 0),
		preloads:   make([]string, 0),
		hooks: hooks{
			beforeCreate: make([]func(any), 0),
			afterCreate:  make([]func(any), 0),
			beforeUpdate: make([]func(any), 0),
			afterUpdate:  make([]func(any), 0),
			beforeDelete: make([]func(any), 0),
			afterDelete:  make([]func(any), 0),
		},
		createData: make(map[string]any),
		updateData: make(map[string]any),
	}
}

// Context устанавливает контекст для запроса
func (a *Aurum[T]) Context(ctx context.Context) *Aurum[T] {
	a.ctx = ctx
	return a
}

// Where добавляет условие WHERE
func (a *Aurum[T]) Where(condition string, args ...any) *Aurum[T] {
	a.conditions = append(a.conditions, condition)
	a.args = append(a.args, args...)
	return a
}

// OrderBy добавляет сортировку
func (a *Aurum[T]) OrderBy(order string) *Aurum[T] {
	a.orderBy = order
	return a
}

// Limit устанавливает лимит записей
func (a *Aurum[T]) Limit(limit int) *Aurum[T] {
	a.limit = limit
	return a
}

// Offset устанавливает смещение
func (a *Aurum[T]) Offset(offset int) *Aurum[T] {
	a.offset = offset
	return a
}

// Preload указывает связанные данные для загрузки
func (a *Aurum[T]) Preload(field string) *Aurum[T] {
	a.preloads = append(a.preloads, field)
	return a
}

// GetById получает запись по ID
func (a *Aurum[T]) GetById(id int) (*T, error) {
	if a.err != nil {
		return nil, a.err
	}

	var entity T
	tableName := getTableName(entity)

	query := fmt.Sprintf("SELECT * FROM %s WHERE id = $1", tableName)
	row := a.db.QueryRowContext(a.ctx, query, id)

	return scanRowIntoEntity[T](row)
}

// GetData получает запись по условию
func (a *Aurum[T]) GetData(field string, value any) (*T, error) {
	if a.err != nil {
		return nil, a.err
	}

	var entity T
	tableName := getTableName(entity)

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", tableName, field)
	row := a.db.QueryRowContext(a.ctx, query, value)

	return scanRowIntoEntity[T](row)
}

// GetAll получает все записи
func (a *Aurum[T]) GetAll() ([]*T, error) {
	if a.err != nil {
		return nil, a.err
	}

	query := a.buildQuery()
	rows, err := a.db.QueryContext(a.ctx, query, a.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRowsIntoEntities[T](rows)
}

// ParamCreate добавляет поле для создания записи
func (a *Aurum[T]) ParamCreate(field string, value any) *Aurum[T] {
	if a.createData == nil {
		a.createData = make(map[string]any)
	}
	a.createData[field] = value
	return a
}

// Create создаёт новую запись
func (a *Aurum[T]) Create() (*T, error) {
	if a.err != nil {
		return nil, a.err
	}

	if len(a.createData) == 0 {
		return nil, fmt.Errorf("no data to create")
	}

	// Выполняем хуки BeforeCreate
	for _, hook := range a.hooks.beforeCreate {
		hook(a.entity)
	}

	var entity T
	tableName := getTableName(entity)

	fields := make([]string, 0, len(a.createData))
	values := make([]any, 0, len(a.createData))
	placeholders := make([]string, 0, len(a.createData))

	i := 1
	for field, value := range a.createData {
		fields = append(fields, field)
		values = append(values, value)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		i++
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(placeholders, ", "),
	)

	row := a.db.QueryRowContext(a.ctx, query, values...)
	result, err := scanRowIntoEntity[T](row)
	if err != nil {
		return nil, err
	}

	// Выполняем хуки AfterCreate
	for _, hook := range a.hooks.afterCreate {
		hook(result)
	}

	return result, nil
}

// Data устанавливает данные для обновления
func (a *Aurum[T]) Data(data map[string]any) *Aurum[T] {
	a.updateData = data
	return a
}

// UpdateData обновляет запись
func (a *Aurum[T]) UpdateData(entity *T, data map[string]any) error {
	if a.err != nil {
		return a.err
	}

	// Выполняем хуки BeforeUpdate
	for _, hook := range a.hooks.beforeUpdate {
		hook(entity)
	}

	var e T
	tableName := getTableName(e)

	fields := make([]string, 0, len(data))
	values := make([]any, 0, len(data))

	i := 1
	for field, value := range data {
		fields = append(fields, fmt.Sprintf("%s = $%d", field, i))
		values = append(values, value)
		i++
	}

	// Добавляем условие по ID
	id := getIDFromEntity(entity)
	fields = append(fields, fmt.Sprintf("id = $%d", i))
	values = append(values, id)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d",
		tableName,
		strings.Join(fields[:len(fields)-1], ", "),
		i,
	)

	_, err := a.db.ExecContext(a.ctx, query, values...)
	if err != nil {
		return err
	}

	// Выполняем хуки AfterUpdate
	for _, hook := range a.hooks.afterUpdate {
		hook(entity)
	}

	return nil
}

// Update обновляет запись по условиям
func (a *Aurum[T]) Update(data map[string]any) error {
	if a.err != nil {
		return a.err
	}

	if len(a.conditions) == 0 {
		return fmt.Errorf("update without WHERE condition is not allowed")
	}

	var entity T
	tableName := getTableName(entity)

	setFields := make([]string, 0, len(data))
	values := make([]any, 0, len(data)+len(a.args))

	i := 1
	for field, value := range data {
		setFields = append(setFields, fmt.Sprintf("%s = $%d", field, i))
		values = append(values, value)
		i++
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		tableName,
		strings.Join(setFields, ", "),
		strings.Join(a.conditions, " AND "),
	)

	values = append(values, a.args...)

	_, err := a.db.ExecContext(a.ctx, query, values...)
	return err
}

// Delete удаляет запись
func (a *Aurum[T]) Delete(entity *T) error {
	if a.err != nil {
		return a.err
	}

	// Выполняем хуки BeforeDelete
	for _, hook := range a.hooks.beforeDelete {
		hook(entity)
	}

	id := getIDFromEntity(entity)
	var e T
	tableName := getTableName(e)

	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", tableName)
	_, err := a.db.ExecContext(a.ctx, query, id)
	if err != nil {
		return err
	}

	// Выполняем хуки AfterDelete
	for _, hook := range a.hooks.afterDelete {
		hook(entity)
	}

	return nil
}

// DeleteById удаляет запись по ID
func (a *Aurum[T]) DeleteById(id int) error {
	if a.err != nil {
		return a.err
	}

	var entity T
	tableName := getTableName(entity)

	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", tableName)
	_, err := a.db.ExecContext(a.ctx, query, id)
	return err
}

// Count возвращает количество записей
func (a *Aurum[T]) Count() (int64, error) {
	if a.err != nil {
		return 0, a.err
	}

	var entity T
	tableName := getTableName(entity)

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	if len(a.conditions) > 0 {
		query += " WHERE " + strings.Join(a.conditions, " AND ")
	}

	var count int64
	err := a.db.QueryRowContext(a.ctx, query, a.args...).Scan(&count)
	return count, err
}

// Exists проверяет существование записи
func (a *Aurum[T]) Exists(id int) (bool, error) {
	if a.err != nil {
		return false, a.err
	}

	var entity T
	tableName := getTableName(entity)

	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)", tableName)
	var exists bool
	err := a.db.QueryRowContext(a.ctx, query, id).Scan(&exists)
	return exists, err
}

// First получает первую запись
func (a *Aurum[T]) First() (*T, error) {
	a.Limit(1)
	results, err := a.GetAll()
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, sql.ErrNoRows
	}
	return results[0], nil
}

// Find получает записи по условиям
func (a *Aurum[T]) Find(conditions map[string]any) ([]*T, error) {
	for field, value := range conditions {
		a.Where(fmt.Sprintf("%s = ?", field), value)
	}
	return a.GetAll()
}

// Transaction выполняет операции в транзакции
func (a *Aurum[T]) Transaction(fn func(tx *Aurum[T]) error) error {
	tx, err := a.db.BeginTx(a.ctx, nil)
	if err != nil {
		return err
	}

	txWrapper := &TxWrapper{tx: tx}
	txOrm := &Aurum[T]{
		db:         txWrapper,
		tableName:  a.tableName,
		ctx:        a.ctx,
		conditions: make([]string, 0),
		args:       make([]any, 0),
		hooks:      a.hooks,
		createData: make(map[string]any),
		updateData: make(map[string]any),
	}

	if err := fn(txOrm); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback error: %v; original error: %w", rbErr, err)
		}
		return err
	}

	return tx.Commit()
}

// BeforeCreate регистрирует хук перед созданием
func (a *Aurum[T]) BeforeCreate(fn func(*T)) *Aurum[T] {
	a.hooks.beforeCreate = append(a.hooks.beforeCreate, func(v any) {
		if entity, ok := v.(*T); ok {
			fn(entity)
		}
	})
	return a
}

// AfterCreate регистрирует хук после создания
func (a *Aurum[T]) AfterCreate(fn func(*T)) *Aurum[T] {
	a.hooks.afterCreate = append(a.hooks.afterCreate, func(v any) {
		if entity, ok := v.(*T); ok {
			fn(entity)
		}
	})
	return a
}

// BeforeUpdate регистрирует хук перед обновлением
func (a *Aurum[T]) BeforeUpdate(fn func(*T)) *Aurum[T] {
	a.hooks.beforeUpdate = append(a.hooks.beforeUpdate, func(v any) {
		if entity, ok := v.(*T); ok {
			fn(entity)
		}
	})
	return a
}

// AfterUpdate регистрирует хук после обновления
func (a *Aurum[T]) AfterUpdate(fn func(*T)) *Aurum[T] {
	a.hooks.afterUpdate = append(a.hooks.afterUpdate, func(v any) {
		if entity, ok := v.(*T); ok {
			fn(entity)
		}
	})
	return a
}

// buildQuery строит SQL запрос
func (a *Aurum[T]) buildQuery() string {
	var entity T
	tableName := getTableName(entity)

	query := fmt.Sprintf("SELECT * FROM %s", tableName)

	if len(a.conditions) > 0 {
		query += " WHERE " + strings.Join(a.conditions, " AND ")
	}

	if a.orderBy != "" {
		query += " ORDER BY " + a.orderBy
	}

	if a.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", a.limit)
	}

	if a.offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", a.offset)
	}

	return query
}

// getTableName получает имя таблицы из тега или названия типа
func getTableName(entity any) string {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		if tag := v.Type().Field(0).Tag.Get("aurum"); tag != "" {
			parts := strings.Split(tag, ",")
			for _, part := range parts {
				if !strings.Contains(part, ";") {
					return part
				}
			}
		}

		typeName := v.Type().Name()
		return strings.ToLower(typeName) + "s"
	}

	return "entities"
}

// getIDFromEntity получает ID из сущности
func getIDFromEntity(entity any) int {
	v := reflect.ValueOf(entity)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		idField := v.FieldByName("ID")
		if idField.IsValid() {
			return int(idField.Int())
		}
		idField = v.FieldByName("Id")
		if idField.IsValid() {
			return int(idField.Int())
		}
		idField = v.FieldByName("id")
		if idField.IsValid() {
			return int(idField.Int())
		}
	}

	return 0
}

// scanRowIntoEntity сканирует строку в сущность
func scanRowIntoEntity[T any](row *sql.Row) (*T, error) {
	var entity T
	v := reflect.ValueOf(&entity).Elem()
	t := v.Type()

	numFields := t.NumField()
	values := make([]any, numFields)
	fieldPointers := make([]any, numFields)

	for i := 0; i < numFields; i++ {
		fieldPointers[i] = reflect.New(t.Field(i).Type).Interface()
		values[i] = fieldPointers[i]
	}

	if err := row.Scan(values...); err != nil {
		return nil, err
	}

	for i := 0; i < numFields; i++ {
		field := v.Field(i)
		ptr := reflect.ValueOf(fieldPointers[i]).Elem()
		if field.CanSet() {
			field.Set(ptr)
		}
	}

	return &entity, nil
}

// scanRowsIntoEntities сканирует строки в сущности
func scanRowsIntoEntities[T any](rows *sql.Rows) ([]*T, error) {
	var entities []*T

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var entity T
		v := reflect.ValueOf(&entity).Elem()
		t := v.Type()

		values := make([]any, len(columns))
		fieldPointers := make([]any, len(columns))

		for i := 0; i < len(columns); i++ {
			fieldPointers[i] = reflect.New(t.Field(i).Type).Interface()
			values[i] = fieldPointers[i]
		}

		if err := rows.Scan(values...); err != nil {
			return nil, err
		}

		for i := 0; i < len(columns); i++ {
			field := v.Field(i)
			ptr := reflect.ValueOf(fieldPointers[i]).Elem()
			if field.CanSet() {
				field.Set(ptr)
			}
		}

		entities = append(entities, &entity)
	}

	return entities, rows.Err()
}

// migrateModel создаёт таблицу для модели
func migrateModel(db *sql.DB, model any) error {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("model must be a struct or pointer to struct")
	}

	t := v.Type()
	tableName := getTableName(model)

	// Для SQLite используем пошаговое создание
	// Сначала создаём таблицу с ID
	createTableQuery := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY AUTOINCREMENT)",
		tableName,
	)
	_, err := db.Exec(createTableQuery)
	if err != nil {
		return err
	}

	// Получаем существующие колонки
	existingColumns := make(map[string]bool)
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid, notnull, pk int
		var name, typ, dflt_value sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt_value, &pk); err != nil {
			continue
		}
		if name.Valid {
			existingColumns[strings.ToLower(name.String)] = true
		}
	}

	// Добавляем колонки по одной
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Name
		fieldType := field.Type

		// Проверяем тег aurum
		if tag := field.Tag.Get("aurum"); tag != "" {
			if tag == "-" {
				continue
			}
			parts := strings.Split(tag, ",")
			if len(parts) > 0 && parts[0] != "" {
				fieldName = parts[0]
			}
		}

		// Пропускаем ID
		if strings.EqualFold(field.Name, "ID") || strings.EqualFold(field.Name, "Id") {
			continue
		}

		// Преобразуем тип Go в тип SQL
		sqlType := getSQLType(fieldType)

		// Проверяем, существует ли колонка
		if existingColumns[strings.ToLower(fieldName)] {
			continue
		}

		// Добавляем колонку
		alterQuery := fmt.Sprintf(
			"ALTER TABLE %s ADD COLUMN %s %s",
			tableName,
			fieldName,
			sqlType,
		)
		_, err := db.Exec(alterQuery)
		if err != nil {
			// Игнорируем ошибку, если колонка уже существует
			if !strings.Contains(err.Error(), "duplicate column") {
				return err
			}
		}
	}

	// Добавляем поля для soft delete и timestamps если их нет
	timestampColumns := map[string]string{
		"deleted_at": "TIMESTAMP NULL",
		"created_at": "TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
		"updated_at": "TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
	}

	for colName, colType := range timestampColumns {
		if !existingColumns[strings.ToLower(colName)] {
			alterQuery := fmt.Sprintf(
				"ALTER TABLE %s ADD COLUMN %s %s",
				tableName,
				colName,
				colType,
			)
			db.Exec(alterQuery) // Игнорируем ошибки
		}
	}

	return nil
}

// getSQLType преобразует тип Go в тип SQL
func getSQLType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "INTEGER"
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.String:
		return "TEXT"
	case reflect.Struct:
		if t == reflect.TypeOf(time.Time{}) {
			return "TIMESTAMP"
		}
		return "TEXT"
	case reflect.Ptr:
		return getSQLType(t.Elem())
	default:
		return "TEXT"
	}
}

// Driver возвращает драйвер для подключения
func Driver() *DatabaseConnection {
	return &DatabaseConnection{}
}

// Postgres создаёт подключение к PostgreSQL
func (dc *DatabaseConnection) Postgres(host, port, user, password, dbname string) error {
	dns := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	return dc.Connection(dns)
}

// Sqlite создаёт подключение к SQLite
func (dc *DatabaseConnection) Sqlite(path string) error {
	conn := Connection{
		Driver:   "sqlite",
		Database: path,
	}
	return dc.ConnectionDeclarative(conn)
}

// Model создаёт экземпляр ORM для работы с моделью
func Model[T any](db DBConnection) *Aurum[T] {
	return New[T](db)
}

// GetDB возвращает DBConnection для DatabaseConnection
func (dc *DatabaseConnection) GetDB() DBConnection {
	return dc.db
}

// SoftDelete выполняет мягкое удаление записи
func (a *Aurum[T]) SoftDelete(id int) error {
	if a.err != nil {
		return a.err
	}

	var entity T
	tableName := getTableName(entity)

	query := fmt.Sprintf("UPDATE %s SET deleted_at = $1 WHERE id = $2", tableName)
	_, err := a.db.ExecContext(a.ctx, query, time.Now(), id)
	return err
}

// WithDeleted включает отображение удалённых записей
func (a *Aurum[T]) WithDeleted() *Aurum[T] {
	// Просто флаг, что нужно игнорировать deleted_at
	// В текущей реализации это просто возвращает тот же экземпляр
	return a
}

// Restore восстанавливает запись после soft delete
func (a *Aurum[T]) Restore(id int) error {
	if a.err != nil {
		return a.err
	}

	var entity T
	tableName := getTableName(entity)

	query := fmt.Sprintf("UPDATE %s SET deleted_at = NULL WHERE id = $1", tableName)
	_, err := a.db.ExecContext(a.ctx, query, id)
	return err
}

// Sum вычисляет сумму по полю
func (a *Aurum[T]) Sum(field string) (float64, error) {
	if a.err != nil {
		return 0, a.err
	}

	var entity T
	tableName := getTableName(entity)

	query := fmt.Sprintf("SELECT SUM(%s) FROM %s", field, tableName)
	if len(a.conditions) > 0 {
		query += " WHERE " + strings.Join(a.conditions, " AND ")
	}

	var result sql.NullFloat64
	err := a.db.QueryRowContext(a.ctx, query, a.args...).Scan(&result)
	if err != nil {
		return 0, err
	}

	if !result.Valid {
		return 0, nil
	}

	return result.Float64, nil
}

// Avg вычисляет среднее значение по полю
func (a *Aurum[T]) Avg(field string) (float64, error) {
	if a.err != nil {
		return 0, a.err
	}

	var entity T
	tableName := getTableName(entity)

	query := fmt.Sprintf("SELECT AVG(%s) FROM %s", field, tableName)
	if len(a.conditions) > 0 {
		query += " WHERE " + strings.Join(a.conditions, " AND ")
	}

	var result sql.NullFloat64
	err := a.db.QueryRowContext(a.ctx, query, a.args...).Scan(&result)
	if err != nil {
		return 0, err
	}

	if !result.Valid {
		return 0, nil
	}

	return result.Float64, nil
}

// GroupBy группирует записи по полям
func (a *Aurum[T]) GroupBy(fields ...string) ([]*T, error) {
	if a.err != nil {
		return nil, a.err
	}

	var entity T
	tableName := getTableName(entity)

	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	if len(a.conditions) > 0 {
		query += " WHERE " + strings.Join(a.conditions, " AND ")
	}

	if len(fields) > 0 {
		query += " GROUP BY " + strings.Join(fields, ", ")
	}

	rows, err := a.db.QueryContext(a.ctx, query, a.args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRowsIntoEntities[T](rows)
}

// BulkCreate создаёт несколько записей за раз
func (a *Aurum[T]) BulkCreate(entities []*T) error {
	if a.err != nil {
		return a.err
	}

	if len(entities) == 0 {
		return nil
	}

	// Выполняем хуки BeforeCreate для каждой сущности
	for _, entity := range entities {
		for _, hook := range a.hooks.beforeCreate {
			hook(entity)
		}
	}

	var entity T
	tableName := getTableName(entity)

	// Получаем имена полей из первой сущности
	v := reflect.ValueOf(entities[0]).Elem()
	t := v.Type()

	fields := make([]string, 0)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := field.Name

		// Пропускаем ID и поля с тегом "-"
		if strings.EqualFold(name, "ID") {
			continue
		}
		if tag := field.Tag.Get("aurum"); tag == "-" {
			continue
		}

		// Пропускаем поля deleted_at, created_at, updated_at
		if strings.EqualFold(name, "DeletedAt") || strings.EqualFold(name, "CreatedAt") || strings.EqualFold(name, "UpdatedAt") {
			continue
		}

		fields = append(fields, getFieldName(field))
	}

	// Строим VALUES для всех сущностей
	valueStrings := make([]string, 0)
	valueArgs := make([]any, 0)
	argIndex := 1

	for _, entity := range entities {
		ev := reflect.ValueOf(entity).Elem()
		values := make([]string, 0)

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			name := field.Name

			if strings.EqualFold(name, "ID") {
				continue
			}
			if tag := field.Tag.Get("aurum"); tag == "-" {
				continue
			}
			if strings.EqualFold(name, "DeletedAt") || strings.EqualFold(name, "CreatedAt") || strings.EqualFold(name, "UpdatedAt") {
				continue
			}

			fieldVal := ev.Field(i).Interface()
			values = append(values, fmt.Sprintf("$%d", argIndex))
			valueArgs = append(valueArgs, fieldVal)
			argIndex++
		}

		valueStrings = append(valueStrings, "("+strings.Join(values, ", ")+")")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		tableName,
		strings.Join(fields, ", "),
		strings.Join(valueStrings, ", "),
	)

	_, err := a.db.ExecContext(a.ctx, query, valueArgs...)
	if err != nil {
		return err
	}

	// Выполняем хуки AfterCreate для каждой сущности
	for _, entity := range entities {
		for _, hook := range a.hooks.afterCreate {
			hook(entity)
		}
	}

	return nil
}

// BulkUpdate обновляет несколько записей за раз
func (a *Aurum[T]) BulkUpdate(entities []*T) error {
	if a.err != nil {
		return a.err
	}

	if len(entities) == 0 {
		return nil
	}

	// Выполняем хуки BeforeUpdate для каждой сущности
	for _, entity := range entities {
		for _, hook := range a.hooks.beforeUpdate {
			hook(entity)
		}
	}

	var entity T
	tableName := getTableName(entity)

	// Строим UPDATE запрос с CASE WHEN
	v := reflect.ValueOf(entities[0]).Elem()
	t := v.Type()

	// Получаем ID всех сущностей
	ids := make([]int, 0, len(entities))
	for _, e := range entities {
		ids = append(ids, getIDFromEntity(e))
	}

	// Для каждого поля строим CASE WHEN
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := field.Name

		if strings.EqualFold(name, "ID") {
			continue
		}
		if tag := field.Tag.Get("aurum"); tag == "-" {
			continue
		}
		if strings.EqualFold(name, "DeletedAt") || strings.EqualFold(name, "CreatedAt") {
			continue
		}

		fieldName := getFieldName(field)

		// Строим CASE WHEN для каждого ID
		caseParts := make([]string, 0)
		args := make([]any, 0)
		argIndex := 1

		for j, e := range entities {
			ev := reflect.ValueOf(e).Elem()
			fieldVal := ev.Field(i).Interface()
			caseParts = append(caseParts, fmt.Sprintf("WHEN $%d THEN $%d", j+1, len(ids)+argIndex))
			args = append(args, fieldVal)
			argIndex++
		}

		query := fmt.Sprintf(
			"UPDATE %s SET %s = CASE id %s END WHERE id = ANY($%d)",
			tableName,
			fieldName,
			strings.Join(caseParts, " "),
			len(ids)+1,
		)

		allArgs := make([]any, 0)
		for _, id := range ids {
			allArgs = append(allArgs, id)
		}
		allArgs = append(allArgs, args...)
		allArgs = append(allArgs, ids)

		_, err := a.db.ExecContext(a.ctx, query, allArgs...)
		if err != nil {
			return err
		}
	}

	// Выполняем хуки AfterUpdate для каждой сущности
	for _, entity := range entities {
		for _, hook := range a.hooks.afterUpdate {
			hook(entity)
		}
	}

	return nil
}

// BulkDelete удаляет несколько записей по ID
func (a *Aurum[T]) BulkDelete(ids []int) error {
	if a.err != nil {
		return a.err
	}

	if len(ids) == 0 {
		return nil
	}

	var entity T
	tableName := getTableName(entity)

	// Строим массив плейсхолдеров
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for i, id := range ids {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, id)
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE id IN (%s)",
		tableName,
		strings.Join(placeholders, ", "),
	)

	_, err := a.db.ExecContext(a.ctx, query, args...)
	return err
}

// Query выполняет произвольный SQL запрос
func (a *Aurum[T]) Query(query string, args ...any) ([]*T, error) {
	if a.err != nil {
		return nil, a.err
	}

	rows, err := a.db.QueryContext(a.ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRowsIntoEntities[T](rows)
}

// Joins добавляет JOIN к запросу
func (a *Aurum[T]) Joins(tables ...string) *Aurum[T] {
	// Сохраняем таблицы для последующего использования в buildQuery
	// В текущей реализации просто добавляем к preloads
	for _, table := range tables {
		a.preloads = append(a.preloads, "join:"+table)
	}
	return a
}

// getFieldName получает имя поля из тега aurum или использует имя поля
func getFieldName(field reflect.StructField) string {
	if tag := field.Tag.Get("aurum"); tag != "" {
		parts := strings.Split(tag, ",")
		if len(parts) > 0 && parts[0] != "" && parts[0] != "-" {
			return parts[0]
		}
	}

	// Конвертируем CamelCase в snake_case
	name := field.Name
	result := make([]byte, 0, len(name)*2)
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, byte(r))
	}
	return strings.ToLower(string(result))
}
