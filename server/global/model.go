package global

import (
	"time"

	"gorm.io/gorm"
)

type GVA_MODEL struct {
	ID        uint           `gorm:"primarykey" json:"ID"` // 主键ID
	CreatedAt time.Time      // 创建时间
	UpdatedAt time.Time      // 更新时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 删除时间
}

// GVA_MODEL_NO_SOFT_DELETE is for legacy tables that do NOT have a deleted_at column.
type GVA_MODEL_NO_SOFT_DELETE struct {
	ID        uint      `gorm:"primarykey;autoIncrement" json:"ID"`
	CreatedAt time.Time // 创建时间
	UpdatedAt time.Time // 更新时间
}
