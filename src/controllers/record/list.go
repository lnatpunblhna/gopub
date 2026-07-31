package recordcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/common"
	"github.com/linclin/gopub/src/library/db"
)

func List(c echo.Context) error {
	ctx := controllers.New(c)
	taskId := ctx.GetString("taskId")
	var records []db.Params
	if common.GetInt(taskId) <= 0 {
		timeNow := ctx.GetString("time")
		records, _ = db.Values("SELECT * FROM `record` where task_id=? and created_at> ? ORDER BY `id` ASC ", taskId, timeNow)
	} else {
		records, _ = db.Values("SELECT * FROM `record` where task_id=? ORDER BY `id` ASC ", taskId)
	}
	return ctx.SetJson(0, records, "")
}
