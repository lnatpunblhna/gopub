package taskcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Task(c echo.Context) error {
	ctx := controllers.New(c)
	// ctx.Task 只在当前用户对其所属项目有权限时才会被加载
	if ctx.Task != nil && ctx.Task.Id != 0 {
		return ctx.SetJson(0, ctx.Task, "")
	}
	// 这个接口在免登录白名单里（免登录看日志页要拿上线单标题、状态、分支等）。
	// 未登录、以及登录了但对该项目没有权限的用户，拿到的副本里不含目标机列表。
	taskId, _ := ctx.GetInt("taskId", 0)
	task, _ := models.GetTaskById(taskId)
	return ctx.SetJson(0, publicTask(task), "")
}

// publicTask 返回上线单的脱敏副本：字段结构保持不变（前端按字段名取值），
// 只抹掉目标机列表；免登录看日志页用的是项目侧的 hosts，不依赖这里。
func publicTask(task *models.Task) *models.Task {
	if task == nil {
		return nil
	}
	t := *task
	t.Hosts = ""
	return &t
}
