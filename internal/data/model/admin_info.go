package model

import "time"

type AdminInfo struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement;comment:ID"`
	Password    string     `gorm:"type:text;not null;comment:密码"`
	LastLoginAt *time.Time `gorm:"type:timestamptz;comment:最后登录时间"`
	DelFlag     int        `gorm:"default:0;comment:删除标志"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"type:timestamptz;autoUpdateTime"`
}

func (AdminInfo) TableName() string {
	return "admin_info"
}
