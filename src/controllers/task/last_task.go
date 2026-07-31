package taskcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/db"
	"github.com/linclin/gopub/src/models"
)

func LastTask(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	var task models.Task
	// 原实现误将整个 Project 结构体作为查询参数传入，这里改为项目 id
	db.QueryRow(&task, "SELECT * FROM task where project_id = ? AND status=3 order by task.id DESC LIMIT 1", ctx.Project.Id)
	return ctx.SetJson(0, task, "")
}
