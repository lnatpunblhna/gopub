package controllers

import (
	"encoding/json"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
	"golang.org/x/crypto/bcrypt"
)

func ChangePasswd(c echo.Context) error {
	ctx := New(c)
	//哈希校验成功后 更新 auth_key
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}

	logger.Info(string(ctx.RequestBody()))

	postData := map[string]string{"newpassword": "", "repeat_newpassword": ""}
	err := json.Unmarshal(ctx.RequestBody(), &postData)
	if err != nil {
		return ctx.SetJson(1, nil, "数据格式错误")
	}

	uid := postData["uid"]
	if common.GetString(ctx.User.Id) != uid && ctx.User.Role != 1 {
		return ctx.SetJson(1, nil, "403")
	}
	newPassword := postData["newpassword"]
	repeatNewpassword := postData["repeat_newpassword"]
	if newPassword == "" || repeatNewpassword == "" {
		return ctx.SetJson(1, nil, "请输入密码")
	}
	var user models.User
	err = db.QueryRow(&user, "SELECT * FROM `user` WHERE id= ?", uid)
	logger.Info(err)
	//验证旧密码

	if newPassword != repeatNewpassword {
		return ctx.SetJson(1, nil, "两次密码输入不一致，请重新输入")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	user.PasswordHash = string(hashedPassword)
	models.UpdateUserById(&user)
	// 改密码后吊销该用户的登录凭据，旧 token 不能继续用。
	// 改的是自己的密码时前端会同步退出登录（见 pages/user/changepasswd.vue）。
	if err := models.RevokeAuthKey(user.Id); err != nil {
		logger.Error("改密码后清理登录凭据失败:", err)
	}
	return ctx.Json(map[string]interface{}{"code": 0, "msg": "sucess"})
}
