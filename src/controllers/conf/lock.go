package confcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Lock(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}
	// ctx.Project 只在当前用户有权限时才会被加载
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "项目不存在或无权限")
	}
	// 1为锁定 0为解锁
	act, _ := ctx.GetInt("act", 0)

	project := ctx.Project

	if act == 1 {
		project.UserLock = int(ctx.User.Id)
	} else {
		project.UserLock = 0
	}

	err := models.UpdateProjectById(project)
	if err != nil {
		return ctx.SetJson(1, nil, err.Error())
	}
	return ctx.SetJson(0, nil, "锁定成功")
}
