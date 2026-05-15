package login

import "github.com/gin-gonic/gin"

type LoginService struct {
}

func NewLoginService() *LoginService {
	return &LoginService{}
}

func (service *LoginService) Login(c *gin.Context) {

}
