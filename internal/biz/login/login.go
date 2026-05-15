package login

import (
	"gorm.io/gorm"
)

type LoginUseCase struct {
	db *gorm.DB
}

func NewLoginUseCase(db *gorm.DB) *LoginUseCase {
	return &LoginUseCase{db: db}
}
func (lu *LoginUseCase) Login(passwd string) {

}
