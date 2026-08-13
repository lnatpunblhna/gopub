package wallecontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/models"
)

func GetMd5(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	url := ctx.GetString("url")
	// url 会被拼进 wget / md5sum 命令，必须先挡住 shell 元字符（components 层还有一道）
	if err := common.ValidDownloadRef(url); err != nil {
		return ctx.SetJson(1, nil, "制品地址不合法—"+err.Error())
	}
	s := components.BaseComponents{}
	s.SetProject(ctx.Project)
	s.SetTask(&models.Task{})
	s.DisableRecord()
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
