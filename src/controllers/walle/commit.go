package wallecontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Commit(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	branch := ctx.GetString("branch")
	// branch 会被拼进本地 git 命令，必须先挡住 shell 元字符（components 层还有一道）
	if err := common.ValidGitRef(branch); err != nil {
		return ctx.SetJson(1, nil, "分支名不合法—"+err.Error())
	}
	s := components.BaseComponents{}
	s.SetProject(ctx.Project)
	s.SetTask(&models.Task{})
	s.DisableRecord()
	g := components.BaseGit{}
	g.SetBaseComponents(s)
	res, err := g.GetCommitList(branch, 25)
	if err != nil {
		return ctx.SetJson(1, nil, "获取Commit错误—"+err.Error())
	}
	return ctx.SetJson(0, res, "")
}
