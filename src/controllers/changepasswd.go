package controllers

import (
	"encoding/json"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/library/common"
	"github.com/linclin/gopub/src/library/db"
	"github.com/linclin/gopub/src/library/logger"
	"github.com/linclin/gopub/src/models"
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
	return ctx.Json(map[string]interface{}{"code": 0, "msg": "sucess"})
}
