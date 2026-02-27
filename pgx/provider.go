package pgx

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ParsePostgresLogLevel(logLevel string) logger.LogLevel {
	switch strings.ToUpper(logLevel) {
	case "INFO":
		// gorm 没有 debug 级别，将 debug 级别映射到 gorm 的 info 级别保证只有 debug 级别输出 sql 日志
		return logger.Warn
	case "WARNING", "WARN":
		return logger.Warn
	case "ERROR":
		return logger.Error
	default:
		return logger.Info
	}
}

func NewPostgres(logLevel string, dsn string, mLogger log.Logger) (*gorm.DB, error) {

	if _, err := url.Parse(dsn); err != nil {
		return nil, fmt.Errorf("invalid dsn: %w", err)
	}

	hl := log.NewHelper(log.With(mLogger, "module", "gorm-postgres"))
	slowLogger := NewGormLogger(
		hl,
		logger.Config{
			SlowThreshold:             time.Second, // 慢查询阈值
			LogLevel:                  ParsePostgresLogLevel(logLevel),
			IgnoreRecordNotFoundError: true,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: slowLogger,
	})
	if err != nil {
		return nil, errors.WithMessage(err, "gorm open failed:")
	}

	// 拿到底层 *sql.DB 进行连接池配置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, errors.WithMessage(err, "gorm get sql db failed:")
	}

	// —— 连接池配置（可根据业务峰值调优） ——
	sqlDB.SetMaxIdleConns(20)                  // 空闲连接数
	sqlDB.SetMaxOpenConns(100)                 // 最大打开连接数
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)  // 空闲连接最大存活时间
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // 连接最大存活时间

	return db, nil
}
