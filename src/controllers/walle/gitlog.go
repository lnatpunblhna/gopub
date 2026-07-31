package wallecontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/components"
	"github.com/linclin/gopub/src/models"
)

func Gitlog(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	s := components.BaseComponents{}
	s.SetProject(ctx.Project)
	s.SetTask(&models.Task{Id: -3})
	err := s.GetGitLog()
	if err != nil {
		return ctx.SetJson(1, nil, "获取GitLog错误—"+err.Error())
	}
	return ctx.SetJson(0, nil, "")
}
