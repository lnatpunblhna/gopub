package taskcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/models"
)

func Task(c echo.Context) error {
	ctx := controllers.New(c)
	taskId, _ := ctx.GetInt("taskId", 0)
	task, _ := models.GetTaskById(taskId)
	return ctx.SetJson(0, task, "")
}
