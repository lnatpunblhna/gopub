package controllers

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
)

func LoginByDocker(c echo.Context) error {
	ctx := New(c)
	if config.RunMode() != "docker" {
		return nil
	}
	jumpUrl := ctx.GetString("jumpurl")
	var user models.User
	err := db.QueryRow(&user, "SELECT * FROM `user` WHERE username= ?", "admin")

	// 凭据缺失或已过期时重新签发，否则 cookie 里带的会是失效的 token
	if err == nil && !models.AuthKeyValid(&user) {
		if err := models.IssueAuthKey(&user); err != nil {
			logger.Error("签发登录凭据失败:", err)
		}
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
