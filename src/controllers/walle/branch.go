package wallecontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Branch(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	s := components.BaseComponents{}
	s.SetProject(ctx.Project)
	s.SetTask(&models.Task{})
	// 只读查询不写 record：错误已经通过接口返回，写库只会把上线日志冲乱、把表撑大
	s.DisableRecord()
	g := components.BaseGit{}
	g.SetBaseComponents(s)
	res, err := g.GetBranchList()
	if err != nil {
		return ctx.SetJson(1, nil, "获取分支错误—"+err.Error())
	}
	return ctx.SetJson(0, res, "")
}
