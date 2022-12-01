package mysql

import (
	"context"
	"github.com/995933447/optionstream"
	"gorm.io/gorm"
)

type OptStreamQuery struct {
	db *gorm.DB
}

var _ optionstream.Queriable = (*OptStreamQuery)(nil)

func NewOptStreamQuery(db *gorm.DB) *OptStreamQuery {
	return &OptStreamQuery{db}
}

func (q *OptStreamQuery) Hit(ctx context.Context, limit, offset int64, list interface{}) (int64, error) {
	db := q.db.WithContext(ctx)
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, err
	}
	if limit > 0 {
		db.Limit(int(limit))
	}
	if offset > 0 {
		db.Limit(int(offset))
	}
	if err := db.Scan(list).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (q *OptStreamQuery) Query(ctx context.Context, limit, offset int64, list interface{}) error {
	db := q.db.WithContext(ctx)
	if limit > 0 {
		db.Limit(int(limit))
	}
	if offset > 0 {
		db.Limit(int(offset))
	}
	if err := db.Scan(list).Error; err != nil {
		return err
	}
	return nil
}

func MakeOnSelectColumnsOptHandler(db *gorm.DB) optionstream.OnOptValStringListProcFunc {
	return func(columns []string) error {
		db.Select(columns)
		return nil
	}
}

