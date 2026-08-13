// 内部接口（前端控制台调用的 /api/get/*、/login 之外的路由）的统一登录校验。
//
// 迁移前每个 handler 各自写 `if ctx.User == nil` 判断，实际只有 12 个 handler 写了，
// 其余 30 多个接口未登录也能读写数据。这里改为在路由层统一挂中间件，
// 白名单（登录入口、免登录看日志页用到的只读接口）在 routers.Register 里显式列出。
package controllers

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/models"
)

// userCtxKey 缓存本次请求解析出的登录用户，避免中间件与 handler 各查一次库
const userCtxKey = "gopub_current_user"

// codeNotLogin 是未登录的业务码：前端 axios 拦截器收到 code===2 就清掉本地
// 凭据并跳回登录页，与原先各 handler 里 SetJson(2, nil, "not login") 保持一致
const codeNotLogin = 2

// roleAdmin 是管理员角色值，与前端 is_admin（Number(user.Role) === 1）
// 及 ChangePasswd 里的 Role != 1 判断保持一致
const roleAdmin int16 = 1

// userCache 让"未登录"（nil）这个结果也能被缓存，避免重复查库
type userCache struct{ user *models.User }

// CurrentUser 返回本次请求的登录用户，未登录返回 nil，结果按请求缓存。
func CurrentUser(c echo.Context) *models.User {
	if cached, ok := c.Get(userCtxKey).(userCache); ok {
		return cached.user
	}
	user := userByToken(c)
	c.Set(userCtxKey, userCache{user: user})
	return user
}

// RequireLogin 统一校验登录态，未登录直接返回 code=2，不再进入 handler。
func RequireLogin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !isLogin(CurrentUser(c)) {
			return notLogin(c)
		}
		return next(c)
	}
}

// RequireAdmin 在登录之外再要求管理员身份，用于建号 / 删号 / 用户列表等接口。
func RequireAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := CurrentUser(c)
		if !isLogin(user) {
			return notLogin(c)
		}
		if user.Role != roleAdmin {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"code": 1, "msg": "无权限操作", "data": nil,
			})
		}
		return next(c)
	}
}

func isLogin(user *models.User) bool {
	return user != nil && user.Id != 0
}

func notLogin(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"code": codeNotLogin, "msg": "not login", "data": nil,
	})
}

// userByToken 解析 `Authorization: TOKEN xxx` 头并查出对应用户
func userByToken(c echo.Context) *models.User {
	ah := c.Request().Header.Get("Authorization")
	if len(ah) <= 5 || strings.ToUpper(ah[0:5]) != "TOKEN" {
		return nil
	}
	token := ah[6:]
	if token == "" {
		return nil
	}
	var users []models.User
	n, err := db.QueryRows(&users, "SELECT * FROM `user` WHERE auth_key= ?", token)
	if n > 0 && err == nil {
		user := &users[0]
		// 已过期（含升级前遗留的无过期时间的永久 token）一律当未登录处理
		if !models.AuthKeyValid(user) {
			return nil
		}
		// 滑动过期：有请求就顺延有效期，闲置超过 authKeyLifetime 才失效
		models.RefreshAuthKey(user)
		return user
	}
	return nil
}
