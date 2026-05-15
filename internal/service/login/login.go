package login

import (
	_ "Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

type LoginService struct {
}

func NewLoginService() *LoginService {
	return &LoginService{}
}

// @Summary 登陆接口
// @Description 登陆接口，只需要接受密码
// @Accept       json
// @Produce      json
// @Param        password   body      string  true  "Password"
// @Success      200 {object} common.Response "登陆成功"
// @Failure      400  {object}  common.SystemError "登陆失败且code = 1000，表示没有初始密码"
// @Router      /login [post]
func (service *LoginService) Login(c *gin.Context) {

}
