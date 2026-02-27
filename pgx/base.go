package pgx

import (
	"time"

	"gorm.io/plugin/soft_delete"
)

type BaseModel struct {
	ID        int64     `gorm:"primarykey"              json:"id"         ch:"id"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime" json:"created_at" ch:"created_at"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime" json:"updated_at" ch:"updated_at"`
}

type BaseModelSoftDelete struct {
	ID        int64                 `gorm:"primarykey"                          json:"id"`
	CreatedAt time.Time             `gorm:"not null;autoCreateTime"             json:"created_at"`
	UpdatedAt time.Time             `gorm:"not null;autoUpdateTime"             json:"updated_at"`
	DeletedAt soft_delete.DeletedAt `gorm:"not null;default:0;softDelete:milli" json:"deleted_at" ch:"deleted_at"`
}
