package taskcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
)

// ErrLog 返回某个上线单的失败原因列表。
//
// task_err_log 这张表一直只写不读：流程级错误（回滚失败、某一步抛错）
// 写进去之后没有任何接口能取出来，页面上自然什么都看不到。
func ErrLog(c echo.Context) error {
	ctx := controllers.New(c)
	// ctx.Task 只在当前用户对其所属项目有权限时才会被加载
	if ctx.Task == nil || ctx.Task.Id == 0 {
		return ctx.SetJson(1, nil, "上线单不存在或无权限")
	}
	rows, err := db.Values(
		"SELECT `id`,`task_id`,`err_info`,`create_time` FROM `task_err_log` WHERE `task_id`=? ORDER BY `id` DESC LIMIT 50",
		ctx.Task.Id)
	if err != nil {
		logger.Error("查询上线错误日志失败 taskId=", ctx.Task.Id, " ", err)
		return ctx.SetJson(1, nil, "查询错误日志失败")
	}
	if rows == nil {
		rows = []db.Params{}
	}
	return ctx.SetJson(0, rows, "")
}
