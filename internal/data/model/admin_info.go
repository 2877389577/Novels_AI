package model

import (
	"time"

	"gorm.io/gorm"
)

type AdminInfo struct {
	gorm.Model
	Password    string     `gorm:"type:text;not null;comment:密码"`
	LastLoginAt *time.Time `gorm:"type:timestamptz;comment:最后登录时间"`
	DelFlag     int        `gorm:"default:0;comment:删除标志"`
}

func (AdminInfo) TableName() string {
	return "admin_info"
}
