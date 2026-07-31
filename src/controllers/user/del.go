package usercontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/db"
	"github.com/linclin/gopub/src/models"
)

func Del(c echo.Context) error {
	ctx := controllers.New(c)
	userId, _ := ctx.GetInt("id", 0)
	if userId == 0 {
		return ctx.SetJson(1, nil, "参数错误")
	}
	err := models.DeleteUser(userId)
	if err != nil {
		return ctx.SetJson(1, nil, err.Error())
	}
	//清理该用户与项目的关联
	db.Exec("DELETE FROM `group` WHERE `user_id` = ? ", userId)
	return ctx.SetJson(0, nil, "删除成功")
}
