package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
)

// Logout 清空当前用户的 auth_key，使已发出的凭据立即失效。
// 由于同一账号多端共用一个 auth_key，登出会让该账号的所有端一起退出。
func Logout(c echo.Context) error {
	ctx := New(c)
	// 未携带有效凭据时无需清理，直接按成功返回，保持前端「登出总是成功」的行为
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(0, nil, "已退出登录")
	}
	if err := models.RevokeAuthKey(ctx.User.Id); err != nil {
		logger.Error("登出清理登录凭据失败:", err)
		return ctx.SetJson(1, nil, "退出登录失败，服务端凭据可能未清理，请重试")
	}
	return ctx.SetJson(0, nil, "已退出登录")
}
