package confcontrollers

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/logger"
	"github.com/linclin/gopub/src/models"
)

func Conf(c echo.Context) error {
	ctx := controllers.New(c)
	projectId, _ := ctx.GetInt("projectId", 0)
	project, _ := models.GetProjectById(projectId)
	return ctx.SetJson(0, project, "")
}

func ConfSave(c echo.Context) error {
	ctx := controllers.New(c)
	//projectId,_:=ctx.GetInt("projectId",0)
	var project models.Project
	if err := json.Unmarshal(ctx.RequestBody(), &project); err != nil {
		logger.Info(err)
		return ctx.NoContent(http.StatusOK)
	}
	logger.Info(models.UpdateProjectById(&project))
	return ctx.NoContent(http.StatusOK)
}
