package confcontrollers

import (
	"encoding/json"
	"time"

	"github.com/astaxie/beego"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/models"
)

type SaveController struct {
	controllers.BaseController
}

func (c *SaveController) Post() {
	//projectId,_:=c.GetInt("projectId",0)
	beego.Info(string(c.Ctx.Input.RequestBody))
	var project models.Project
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &project)
	if err != nil {
		c.SetJson(1, nil, "数据格式错误")
		return
	}
	now := time.Now()
	if project.Id != 0 {
		oldProject, err := models.GetProjectById(project.Id)
		if err != nil {
			c.SetJson(1, nil, "项目不存在")
			return
		}
		project.UserId = oldProject.UserId
		if project.UserId == 0 && c.User != nil && c.User.Id != 0 {
			project.UserId = uint(c.User.Id)
		}
		project.CreatedAt = oldProject.CreatedAt
		if project.CreatedAt.IsZero() {
			project.CreatedAt = now
		}
		project.UpdatedAt = now
		err = models.UpdateProjectById(&project)
	} else {
		if c.User == nil || c.User.Id == 0 {
			c.SetJson(2, nil, "not login")
			return
		}
		project.UserId = uint(c.User.Id)
		project.CreatedAt = now
		project.UpdatedAt = now
		_, err = models.AddProject(&project)
	}
	if err != nil {
		c.SetJson(1, nil, "数据库更新错误")
		return
	}
	c.SetJson(0, project, "修改成功")
	return
}
