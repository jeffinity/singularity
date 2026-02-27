package migratex

import (
	"context"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type GormModel interface {
	TableName() string
}

type Migrator struct {
	pg    *gorm.DB
	tbMap map[string]any
}

func NewAllMigrator(pg *gorm.DB, allModels []any) *Migrator {

	tbMap := make(map[string]any)
	for _, model := range allModels {
		if tb, ok := model.(GormModel); ok {
			tbMap[tb.TableName()] = tb
		}
	}

	return &Migrator{
		pg:    pg,
		tbMap: tbMap,
	}
}

func (m *Migrator) ListTables() (tbs []string) {

	for name := range m.tbMap {
		tbs = append(tbs, name)
	}
	return
}

func (m *Migrator) MigrateAll(ctx context.Context) error {
	for _, model := range m.tbMap {
		err := m.pg.WithContext(ctx).AutoMigrate(model)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) MigrateOne(ctx context.Context, tbName string) error {

	if tb, ok := m.tbMap[tbName]; ok {
		return errors.WithStack(m.pg.WithContext(ctx).AutoMigrate(tb))
	}
	return nil
}
