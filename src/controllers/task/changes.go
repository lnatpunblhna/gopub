package taskcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/components"
	"github.com/linclin/gopub/src/library/db"
	"github.com/linclin/gopub/src/models"
)

func Changes(c echo.Context) error {
	ctx := controllers.New(c)
	taskId, _ := ctx.GetInt("taskId", 0)

	if taskId == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}

	var task models.Task
	db.QueryRow(&task, "SELECT * FROM task where task.id = ?", taskId)

	project, err := models.GetProjectById(task.ProjectId)
	if err != nil || project == nil {
		return ctx.SetJson(1, nil, "Parameter error")
	}

	if project.RepoType != "git" {
		return ctx.SetJson(1, nil, "Project is not git")
	}

	var last_task models.Task
	db.QueryRow(&last_task, "SELECT * FROM task where project_id = ? AND status=3 order by task.id DESC LIMIT 1", task.ProjectId)

	s := components.BaseComponents{}
	s.SetProject(project)
	s.SetTask(&task)

	g := components.BaseGit{}
	g.SetBaseComponents(s)
	files, _ := g.DiffBetweenCommits(task.Branch, task.CommitId, last_task.CommitId)

	var fileinfos []map[string]string
	if len(files) < 10 && len(files) > 0 {
		for _, filepath := range files {
			fileinfo, _ := g.GetLastModifyInfo(task.Branch, filepath)
			fileinfo["path"] = filepath
			fileinfos = append(fileinfos, fileinfo)
		}
	}
	return ctx.SetJson(0, fileinfos, "")
}
