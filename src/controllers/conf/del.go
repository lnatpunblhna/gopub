package confcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Del(c echo.Context) error {
	ctx := controllers.New(c)
	// ctx.Project 只在当前用户有权限时才会被加载，为 nil 即无权或项目不存在
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "项目不存在或无权限")
	}
	err := models.DeleteProject(ctx.Project.Id)
	if err != nil {
		return ctx.SetJson(1, nil, err.Error())
	}
	return ctx.SetJson(0, nil, "删除成功")
}
