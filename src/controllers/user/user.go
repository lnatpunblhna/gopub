package usercontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/db"
)

func User(c echo.Context) error {
	ctx := controllers.New(c)
	userId, _ := ctx.GetInt("id")
	if userId == 0 {
		users, _ := db.Values("SELECT * FROM `user` ")
		return ctx.SetJson(0, users, "")
	}
	var res db.Params
	users, err := db.Values("SELECT * FROM `user` where id = ? ", userId)
	if err == nil && len(users) > 0 {
		res = users[0]
	}
	return ctx.SetJson(0, res, "")
}
