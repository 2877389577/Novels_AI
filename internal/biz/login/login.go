package login

import (
	"context"
	"errors"
	"log/slog"

	"Novels_AI/backend/internal/data"
	"Novels_AI/backend/internal/pkg/common"

	"golang.org/x/crypto/bcrypt"
)

type LoginUseCase struct {
	adminInfoData AdminInfoRepo
}

type AdminInfoRepo interface {
	FirstActive(ctx context.Context) (*data.AdminInfo, error)
	Create(ctx context.Context, passwd string) error
	SetPassword(ctx context.Context, passwd string) error
}

func NewLoginUseCase(adminInfoData AdminInfoRepo) *LoginUseCase {
	return &LoginUseCase{adminInfoData: adminInfoData}
}

// IsPasswordInitialized 检查管理员初始化密码是否已经设置。
func (lu *LoginUseCase) IsPasswordInitialized(ctx context.Context) (bool, error) {
	admin, err := lu.adminInfoData.FirstActive(ctx)
	if errors.Is(err, data.ErrAdminInfoNotFound) {
		return false, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "IsPasswordInitialized failed", "err", err)
		return false, err
	}

	return admin.Password != "", nil
}

// SetPassword 仅在管理员密码为空时写入首次初始化密码。
func (lu *LoginUseCase) SetPassword(ctx context.Context, passwd string) error {
	admin, err := lu.adminInfoData.FirstActive(ctx)
	if err != nil && !errors.Is(err, data.ErrAdminInfoNotFound) {
		return err
	}
	if err == nil && admin.Password != "" {
		return common.PasswordAlreadySet
	}

	adminNotFound := errors.Is(err, data.ErrAdminInfoNotFound)
	hashedPassword, err := hashPassword(passwd)
	if err != nil {
		slog.ErrorContext(ctx, "hashing password failed", "err", err)
		return err
	}

	if adminNotFound {
		return lu.adminInfoData.Create(ctx, hashedPassword)
	}

	return lu.adminInfoData.SetPassword(ctx, hashedPassword)
}

// Login 校验初始化密码，密码正确时允许本次登录通过。
func (lu *LoginUseCase) Login(ctx context.Context, passwd string) error {
	admin, err := lu.adminInfoData.FirstActive(ctx)
	if errors.Is(err, data.ErrAdminInfoNotFound) {
		return common.NoInitialPassword
	}
	if err != nil {
		slog.ErrorContext(ctx, "Login failed", "err", err)
		return err
	}
	if admin.Password == "" {
		return common.NoInitialPassword
	}
	if !checkPassword(admin.Password, passwd) {
		return common.PasswordIncorrect
	}

	return nil
}

// hashPassword 使用 bcrypt 保存密码，避免数据库中出现明文密码。
func hashPassword(passwd string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func checkPassword(hashedPassword, passwd string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(passwd)) == nil
}
