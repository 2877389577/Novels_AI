package login

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"Novels_AI/backend/internal/data/dto"
	"Novels_AI/backend/internal/pkg/common"

	"github.com/gin-gonic/gin"
)

type LoginService struct {
	useCase LoginUseCase
}

type LoginUseCase interface {
	IsPasswordInitialized(ctx context.Context) (bool, error)
	SetPassword(ctx context.Context, params dto.PasswordRequest) error
	Login(ctx context.Context, params dto.PasswordRequest) error
}

func NewLoginService(useCase LoginUseCase) *LoginService {
	return &LoginService{useCase: useCase}
}

// @Summary 检查初始化密码
// @Description 检查管理员初始化密码是否已经设置
// @Tags login
// @Produce json
// @Success 200 {object} common.Response{data=map[string]bool} "code = 0 表示检查成功;data.initialized 表示是否已设置初始化密码"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /login/initial-password [get]
func (service *LoginService) CheckInitialPassword(c *gin.Context) {
	initialized, err := service.useCase.IsPasswordInitialized(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
		Data: gin.H{"initialized": initialized},
	})
}

// @Summary 设置初始化密码
// @Description 仅在管理员密码为空时允许设置初始化密码
// @Tags login
// @Accept json
// @Produce json
// @Param password body dto.PasswordRequest true "Password"
// @Success 200 {object} common.Response "code = 0，表示设置成功;code = 1001，表示密码已设置"
// @Failure 400 {object} common.Response "请求参数错误"
// @Failure 500 {object} common.SystemError "系统错误"
// @Router /login/password [post]
func (service *LoginService) SetPassword(c *gin.Context) {
	request, ok := bindPassword(c)
	if !ok {
		return
	}

	if err := service.useCase.SetPassword(c.Request.Context(), request); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "成功",
	})
}

// @Summary 登陆接口
// @Description 登陆接口，只需要接受密码
// @Tags login
// @Accept       json
// @Produce      json
// @Param        password   body      dto.PasswordRequest true "密码"
// @Success      200 {object} common.Response "code = 0，表示登陆成功; code = 1000，表示没有初始密码,登陆失败"
// @Failure      500  {object}  common.SystemError "系统错误"
// @Router      /login [post]
func (service *LoginService) Login(c *gin.Context) {
	request, ok := bindPassword(c)
	if !ok {
		return
	}

	if err := service.useCase.Login(c.Request.Context(), request); err != nil {
		_ = c.Error(err)
		return
	}

	sessionID, err := newSessionID()
	if err != nil {
		_ = c.Error(err)
		return
	}

	err = common.SetSession(c, common.SessionKey, sessionID)
	if err != nil {
		_ = c.Error(err)
		return
	}
	// 不设置 MaxAge/Expires，让浏览器把它作为会话 Cookie，关闭窗口后登录态失效。
	/*c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookieName, sessionID, 0, "/", "", false, true)*/
	c.JSON(http.StatusOK, &common.Response{
		Code: 0,
		Msg:  "登陆成功",
	})
}

func bindPassword(c *gin.Context) (dto.PasswordRequest, bool) {
	var request dto.PasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, &common.Response{
			Code: http.StatusBadRequest,
			Msg:  "请求参数错误",
		})
		return dto.PasswordRequest{}, false
	}

	return request, true
}

func newSessionID() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes[:]), nil
}
