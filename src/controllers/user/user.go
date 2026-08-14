package usercontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/db"
)

// userColumns 是允许下发给前端的字段。
//
// 原实现是 SELECT *，会把 password_hash、password_reset_token、
// email_confirmation_token 一并返回，其中 auth_key 就是登录凭据本身——
// 拿到别人的 auth_key 就能直接冒充对方发请求。前端只用到这里列出的字段：
// 用户列表页要 id/realname/username/email/role/from_ldap/created_at/updated_at，
// 改密码页要 id/username/email/realname，改用户页再多一个 from_ldap。
const userColumns = "`id`,`username`,`realname`,`email`,`role`,`from_ldap`,`status`,`avatar`,`created_at`,`updated_at`"

// User 查用户信息。
//
// 这个接口挂在 RequireLogin 之后而非 RequireAdmin：普通用户改自己密码时，
// changepasswd 页面要用它显示自己的用户名 / 邮箱 / 花名。权限在这里分两档：
// 不带 id 的「拉全部用户」只有管理员能用，带 id 的管理员可查任何人、
// 普通用户只能查自己。
func User(c echo.Context) error {
	ctx := controllers.New(c)
	userId, _ := ctx.GetInt("id")
	admin := controllers.IsAdmin(ctx.User)

	if userId == 0 {
		if !admin {
			return ctx.SetJson(1, nil, "无权限操作")
		}
		users, _ := db.Values("SELECT " + userColumns + " FROM `user` ")
		return ctx.SetJson(0, users, "")
	}

	if !admin && (ctx.User == nil || ctx.User.Id != userId) {
		return ctx.SetJson(1, nil, "无权限操作")
	}
	var res db.Params
	users, err := db.Values("SELECT "+userColumns+" FROM `user` where id = ? ", userId)
	if err == nil && len(users) > 0 {
		res = users[0]
	}
	return ctx.SetJson(0, res, "")
}
