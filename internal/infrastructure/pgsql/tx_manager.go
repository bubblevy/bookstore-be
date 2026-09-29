package pgsql

import (
	"context"

	"github.com/bubblevy/bookstore/internal/repository"
	"gorm.io/gorm"
)

type txKey struct{}

type txManager struct {
	db *gorm.DB
}

func NewTransactionManager(db *gorm.DB) repository.TransactionManager {
	return &txManager{db: db}
}

func (t *txManager) WithTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey{}, tx)
		return fn(txCtx)
	})
}

func extractDB(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return db.WithContext(ctx)
}
