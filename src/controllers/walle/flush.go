package wallecontrollers

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Flush(c echo.Context) error {
	ctx := controllers.New(c)
	projectIds := ctx.GetString("projectIds")
	projectIdsArr := strings.Split(projectIds, ",")
	res := []map[string]interface{}{}
	for _, projectId := range projectIdsArr {
		Project, err := models.GetProjectById(common.GetInt(projectId))
		if err != nil {
			continue
		}
		// 这里会在目标机上执行刷新命令，逐个项目校验权限，跳过无权的
		if !controllers.CanAccessProject(ctx.User, Project) {
			res = append(res, map[string]interface{}{"name": Project.Name, "err": "无权限操作该项目"})
			continue
		}
		s := components.BaseComponents{}
		s.SetProject(Project)
		s.SetTask(&models.Task{})
		s.SetScope(models.RecordScopeFlush)
		s.SetOperatorFromUser(ctx.User)
		err = s.GetExecFlush()
		if err != nil {
			res = append(res, map[string]interface{}{"name": Project.Name, "err": err.Error()})
		} else {
			res = append(res, map[string]interface{}{"name": Project.Name, "msg": "success"})
		}
	}
	return ctx.SetJson(0, res, "")
}
