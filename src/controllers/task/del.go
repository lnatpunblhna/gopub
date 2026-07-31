package taskcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/models"
)

func Del(c echo.Context) error {
	ctx := controllers.New(c)
	taskId, _ := ctx.GetInt("taskId", 0)
	err := models.DeleteTask(taskId)
	if err != nil {
		return ctx.SetJson(1, nil, err.Error())
	}
	return ctx.SetJson(0, nil, "删除成功")
}
