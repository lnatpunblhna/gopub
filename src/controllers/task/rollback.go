package taskcontrollers

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/models"
)

func RollBack(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Task == nil || ctx.Task.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	project, _ := models.GetProjectById(ctx.Task.ProjectId)
	if project == nil || project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	ctx.Project = project
	//上线成功 以及审核失败不允许上线
	if ctx.Task.Status != 3 {
		return ctx.SetJson(1, nil, "此任务未完成")
	}
	//正在上线的不允许上线
	if ctx.Task.Action == 1 {
		return ctx.SetJson(1, nil, "此任务为回滚项目")
	}
	//不允许回滚项目
	if ctx.Task.EnableRollback == 0 {
		return ctx.SetJson(1, nil, "此任务为不允许回滚")
	}
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}
	var task models.Task
	task.Action = 1
	task.EnableRollback = 0
	task.Branch = ctx.Task.Branch
	task.CommitId = ctx.Task.CommitId
	task.Title = ctx.Task.Title + "-回滚项目"
	task.FileList = ctx.Task.FileList
	task.ExLinkId = ctx.Task.ExLinkId
	if ctx.GetString("this") == "this" {
		task.LinkId = ctx.Task.LinkId
	} else {
		task.LinkId = ctx.Task.ExLinkId
	}
	task.ProjectId = ctx.Task.ProjectId
	task.Status = 0
	task.UserId = uint(ctx.User.Id)
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	task.Hosts = ctx.Task.Hosts
	_, err := models.AddTask(&task)
	if err != nil {
		return ctx.SetJson(1, nil, "数据库更新错误")
	}
	return ctx.SetJson(0, task, "创建成功")
}
