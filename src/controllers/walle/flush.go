package wallecontrollers

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/common"
	"github.com/linclin/gopub/src/library/components"
	"github.com/linclin/gopub/src/models"
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
		s := components.BaseComponents{}
		s.SetProject(Project)
		s.SetTask(&models.Task{Id: -2})
		err = s.GetExecFlush()
		if err != nil {
			res = append(res, map[string]interface{}{"name": Project.Name, "err": err.Error()})
		} else {
			res = append(res, map[string]interface{}{"name": Project.Name, "msg": "success"})
		}
	}
	return ctx.SetJson(0, res, "")
}
