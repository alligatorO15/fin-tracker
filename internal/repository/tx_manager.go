package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// менеджр транзакций управляет транзакциями БД
type TxManager interface {
	// WithTx оборачивает репу-метод и выполняет функцию внутри транзакции
	// Если функци возвращает ошибку - транзакция откатывается
	// Если функция завершается успешно - транзакция коммитится
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// DBTX(database and transaction execut) единый интерфейс для работы с бд (его реализует и pool, и tx)
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

type txManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) TxManager {
	return &txManager{pool: pool}
}

// txKey ключ для хранения транзакции в бд(позволяет избежать коллизий имен)
type txKey struct{}

// WithTx выполняет последующие репо-методы внутри транзакции, обеспечивает атомарность операций(транзакции)
func (m *txManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	// Проверяем есть ли транзакция в ctx.Value
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx) // Если внутри контекста уже есть транзакция просто выполняем репо-методы
	}

	// Если нет, то начинаем новую транзакцию
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}

	// Помещаем транзакцию в контекст
	txCtx := context.WithValue(ctx, txKey{}, tx)

	// Выполняем функцию
	if err := fn(txCtx); err != nil {
		// При ошибке откатываем
		_ = tx.Rollback(ctx)
		return err
	}

	// При успехе коммитим все изменения
	return tx.Commit(ctx)
}

// GetTxOrPool возвращает либо pool, либо tx из контекста
func GetTxOrPool(ctx context.Context, pool *pgxpool.Pool) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}
