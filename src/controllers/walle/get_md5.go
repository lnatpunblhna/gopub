package wallecontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/components"
	"github.com/linclin/gopub/src/models"
)

func GetMd5(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	url := ctx.GetString("url")
	s := components.BaseComponents{}
	s.SetProject(ctx.Project)
	s.SetTask(&models.Task{})
	f := components.BaseFile{}
	f.SetBaseComponents(s)
	err := f.UpdateRepo(url, "")
	if err != nil {
		return ctx.SetJson(1, nil, "获取md5错误—"+err.Error())
	}
	res, err := f.CheckFiles(url, "")
	if err != nil {
		return ctx.SetJson(1, nil, "获取md5错误—"+err.Error())
	}
	return ctx.SetJson(0, res, "")
}
