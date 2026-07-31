package taskcontrollers

import (
	"encoding/json"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/common"
	"github.com/linclin/gopub/src/library/components"
	"github.com/linclin/gopub/src/library/logger"
	"github.com/linclin/gopub/src/models"
)

func Save(c echo.Context) error {
	ctx := controllers.New(c)
	//projectId,_:=ctx.GetInt("projectId",0)
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}
	logger.Info(string(ctx.RequestBody()))
	var task models.Task
	err := json.Unmarshal(ctx.RequestBody(), &task)
	if err != nil {
		return ctx.SetJson(1, nil, "数据库更新错误"+err.Error())
	}
	if task.Id != 0 {
		err = models.UpdateTaskById(&task)
	} else {
		task.UserId = uint(ctx.User.Id)
		task.CreatedAt = time.Now()
		task.UpdatedAt = time.Now()
		task.EnableRollback = 1
		if task.Hosts == "" {
			ss, errPro := models.GetProjectById(task.ProjectId)
			if errPro == nil && ss != nil {
				task.Hosts = ss.Hosts
				task.HostGroup = ss.HostGroup
				// 分批发布：按服务器分组拆成多个任务
				if ss.IsGroup == 1 {
					s := components.BaseComponents{}
					s.SetProject(ss)
					mapHost := s.GetGroupHost()
					for k, v := range mapHost {
						task1 := task
						task1.Hosts = v
						task1.Title = task1.Title + "第" + common.GetString(k) + "批"
						models.AddTask(&task1)
					}
					return ctx.SetJson(0, task, "修改成功")
				}
			}
		}
		_, err = models.AddTask(&task)
	}
	if err != nil {
		return ctx.SetJson(1, nil, "数据库更新错误")
	}
	return ctx.SetJson(0, task, "修改成功")
}
