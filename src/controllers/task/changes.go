package taskcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Changes(c echo.Context) error {
	ctx := controllers.New(c)
	// ctx.Task 只在当前用户对其所属项目有权限时才会被加载
	if ctx.Task == nil || ctx.Task.Id == 0 {
		return ctx.SetJson(1, nil, "上线单不存在或无权限")
	}
	task := *ctx.Task

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
	// 版本对比是只读操作，却带着真实 taskId 在往 record 里写：
	// 每打开一次上线页，进度日志里就会多出一批 git diff 命令
	s.DisableRecord()

	g := components.BaseGit{}
	g.SetBaseComponents(s)
	files, _ := g.DiffBetweenCommits(task.Branch, task.CommitId, last_task.CommitId)

	var fileinfos []map[string]string
	if len(files) < 10 && len(files) > 0 {
		for _, filepath := range files {
			// 取不到修改信息时返回的是 nil map，直接赋值会 panic
			fileinfo, err := g.GetLastModifyInfo(task.Branch, filepath)
			if err != nil || fileinfo == nil {
				continue
			}
			fileinfo["path"] = filepath
			fileinfos = append(fileinfos, fileinfo)
		}
	}
	return ctx.SetJson(0, fileinfos, "")
}
