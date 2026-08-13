package confcontrollers

import (
	"encoding/json"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Save(c echo.Context) error {
	ctx := controllers.New(c)
	//projectId,_:=ctx.GetInt("projectId",0)
	logger.Info(string(ctx.RequestBody()))
	var project models.Project
	err := json.Unmarshal(ctx.RequestBody(), &project)
	if err != nil {
		return ctx.SetJson(1, nil, "数据格式错误")
	}
	now := time.Now()
	if project.Id != 0 {
		oldProject, err := models.GetProjectById(project.Id)
		if err != nil {
			return ctx.SetJson(1, nil, "项目不存在")
		}
		// 必须拿库里的旧记录判断权限：请求体里的 level 等字段可被伪造，
		// 用它判断的话 Role=10 只需把 level 填成 2 就能改任意项目
		if !controllers.CanAccessProject(ctx.User, oldProject) {
			return ctx.SetJson(1, nil, "无权限操作该项目")
		}
		project.UserId = oldProject.UserId
		if project.UserId == 0 && ctx.User != nil && ctx.User.Id != 0 {
			project.UserId = uint(ctx.User.Id)
		}
		project.CreatedAt = oldProject.CreatedAt
		if project.CreatedAt.IsZero() {
			project.CreatedAt = now
		}
		project.UpdatedAt = now
		err = models.UpdateProjectById(&project)
		if err != nil {
			return ctx.SetJson(1, nil, "数据库更新错误")
		}
	} else {
		if ctx.User == nil || ctx.User.Id == 0 {
			return ctx.SetJson(2, nil, "not login")
		}
		project.UserId = uint(ctx.User.Id)
		project.CreatedAt = now
		project.UpdatedAt = now
		if _, err = models.AddProject(&project); err != nil {
			return ctx.SetJson(1, nil, "数据库更新错误")
		}
	}
	return ctx.SetJson(0, project, "修改成功")
}
