package wallecontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Tag(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	s := components.BaseComponents{}
	s.SetProject(ctx.Project)
	s.SetTask(&models.Task{})
	s.DisableRecord()
	g := components.BaseGit{}
	g.SetBaseComponents(s)
	res, err := g.GetTagList(200)
	if err != nil {
		return ctx.SetJson(1, nil, "获取tag错误—"+err.Error())
	}
	return ctx.SetJson(0, res, "")
}
