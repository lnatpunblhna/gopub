package taskcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Del(c echo.Context) error {
	ctx := controllers.New(c)
	// ctx.Task 只在当前用户对其所属项目有权限时才会被加载
	if ctx.Task == nil || ctx.Task.Id == 0 {
		return ctx.SetJson(1, nil, "上线单不存在或无权限")
	}
	err := models.DeleteTask(ctx.Task.Id)
	if err != nil {
		return ctx.SetJson(1, nil, err.Error())
	}
	return ctx.SetJson(0, nil, "删除成功")
}
