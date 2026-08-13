package taskcontrollers

import (
	"encoding/json"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
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
		// 改已有上线单：按库里那张单所属的项目判断权限，
		// 不能信请求体里的 ProjectId（伪造成自己有权的项目就绕过去了）
		oldTask, errTask := models.GetTaskById(task.Id)
		if errTask != nil || oldTask == nil {
			return ctx.SetJson(1, nil, "上线单不存在")
		}
		if !controllers.CanAccessTask(ctx.User, oldTask) {
			return ctx.SetJson(1, nil, "无权限操作该上线单")
		}
		// 归属字段一律以库里的为准：否则可以把自己有权的单改挂到别人的项目下，
		// 再触发发布，等于绕开上面的权限判断
		task.ProjectId = oldTask.ProjectId
		task.UserId = oldTask.UserId
		err = models.UpdateTaskById(&task)
	} else {
		// 新建上线单：必须对目标项目有权限
		if !controllers.CanAccessProjectId(ctx.User, task.ProjectId) {
			return ctx.SetJson(1, nil, "项目不存在或无权限")
		}
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
