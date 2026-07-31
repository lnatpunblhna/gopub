package controllers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/library/common"
	"github.com/linclin/gopub/src/library/config"
	"github.com/linclin/gopub/src/library/db"
	"github.com/linclin/gopub/src/models"
)

func LoginByDocker(c echo.Context) error {
	ctx := New(c)
	if config.RunMode() != "docker" {
		return nil
	}
	jumpUrl := ctx.GetString("jumpurl")
	var user models.User
	err := db.QueryRow(&user, "SELECT * FROM `user` WHERE username= ?", "admin")

	if err == nil && user.AuthKey == "" {
		userAuth := common.Md5String(user.Username + common.GetString(time.Now().Unix()))
		user.AuthKey = userAuth
		models.UpdateUserById(&user)
	}
	resUserInfo := map[string]interface{}{"user": user, "login": true}
	userInfoJson, _ := json.Marshal(resUserInfo)
	ctx.SetCookie(&http.Cookie{
		Name:   "gopub_userinfo",
		Value:  url.QueryEscape(string(userInfoJson)),
		MaxAge: 3600 * 24 * 2,
		Path:   "/",
	})
	return ctx.Redirect(http.StatusFound, jumpUrl)
}
