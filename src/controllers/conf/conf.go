package confcontrollers

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Conf(c echo.Context) error {
	ctx := controllers.New(c)
	// ctx.Project 只在当前用户对该项目有权限时才会被加载，此时才下发完整配置
	if ctx.Project != nil && ctx.Project.Id != 0 {
		return ctx.SetJson(0, ctx.Project, "")
	}
	// 这个接口在免登录白名单里（免登录看日志页要拿项目名和环境）。
	// 未登录、以及登录了但对该项目没有权限的用户，都只拿到页面展示用的字段，
	// 仓库口令、服务器列表、部署命令等一概不返回
	projectId, _ := ctx.GetInt("projectId", 0)
	project, _ := models.GetProjectById(projectId)
	return ctx.SetJson(0, publicProject(project), "")
}

// publicProject 返回项目的脱敏副本：保留字段结构不变（前端按字段名取值），
// 只填充展示所需的几项，其余留零值。
func publicProject(project *models.Project) *models.Project {
	if project == nil {
		return nil
	}
	return &models.Project{
		Id:       project.Id,
		Name:     project.Name,
		Level:    project.Level,
		Status:   project.Status,
		RepoType: project.RepoType,
	}
}

func ConfSave(c echo.Context) error {
	ctx := controllers.New(c)
	//projectId,_:=ctx.GetInt("projectId",0)
	var project models.Project
	if err := json.Unmarshal(ctx.RequestBody(), &project); err != nil {
		logger.Info(err)
		return ctx.NoContent(http.StatusOK)
	}
	// 按库里的旧记录判断权限，不能信请求体里的字段
	if !controllers.CanAccessProjectId(ctx.User, project.Id) {
		return ctx.SetJson(1, nil, "项目不存在或无权限")
	}
	logger.Info(models.UpdateProjectById(&project))
	return ctx.NoContent(http.StatusOK)
}
