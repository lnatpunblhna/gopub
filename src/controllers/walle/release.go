package wallecontrollers

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/common"
	"github.com/linclin/gopub/src/library/components"
	"github.com/linclin/gopub/src/library/db"
	"github.com/linclin/gopub/src/models"
)

// releaseJob 承载一次上线/回滚任务所需的数据。
//
// 上线流程在独立 goroutine 中执行，而 Echo 会复用 echo.Context 对象，
// 请求结束后继续持有它会读到其他请求的数据，因此这里只保留 Task/Project，
// 不引用请求上下文。
type releaseJob struct {
	Task    *models.Task
	Project *models.Project
}

func Release(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Task == nil || ctx.Task.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	project, _ := models.GetProjectById(ctx.Task.ProjectId)
	if project == nil || project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	ctx.Project = project
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}
	//不是自己的项目不允许上线(除非是管理员项目)
	if ctx.User.Id != int(ctx.Task.UserId) && ctx.User.Id != 1 && ctx.Task.UserId != 1 {
		return ctx.SetJson(1, nil, "not such uer")
	}
	//上线成功 以及审核失败不允许上线
	if ctx.Task.Status == 2 || ctx.Task.Status == 3 {
		return ctx.SetJson(1, nil, "此项目已完成")
	}
	//正在上线的不允许上线
	if ctx.Task.IsRun == 1 {
		return ctx.SetJson(1, nil, "此项目正在上线中")
	}
	//删除上线日志记录
	db.Exec("DELETE FROM `record` WHERE `task_id`= ? ", ctx.Task.Id)

	job := &releaseJob{Task: ctx.Task, Project: project}
	if job.Task.Action == 0 {
		//生成版本号
		job.makeVersion()
		go func() {
			err := job.releaseHandling()
			if err != nil {
				models.AddTaskErrLog(&models.TaskErrLog{
					ErrInfo: err.Error(),
					TaskId:  job.Task.Id,
				})
			}
		}()
	} else {
		go job.rollBackHandling()
	}
	return ctx.SetJson(0, nil, "")
}

func (c *releaseJob) makeVersion() {
	c.Task.IsRun = 1
	version := time.Now().Format("20060102-150405")
	c.Task.LinkId = version
	c.Task.UpdatedAt = time.Now()
	models.UpdateTaskById(c.Task)
}

// 回滚任务
func (c *releaseJob) rollBackHandling() error {
	s := components.BaseComponents{}
	s.SetProject(c.Project)
	s.SetTask(c.Task)
	g := components.BaseGit{}
	g.SetBaseComponents(s)
	err := s.UpdateRemoteServers(c.Task.LinkId)
	if err != nil {
		c.failHandling(&s)
		return err
	}
	err = c.changeReleaseData()
	if err != nil {
		return err
	}
	return nil
}

// 普通上线任务
func (c *releaseJob) updateRecord(action int) error {
	_, err := db.Exec("UPDATE `record` SET `action`= ?  WHERE`task_id` = ? and action=0", action, c.Task.Id)
	if err != nil {
		return err
	}
	return nil
}

// 普通上线任务
func (c *releaseJob) releaseHandling() error {
	s := components.BaseComponents{}
	s.SetProject(c.Project)
	s.SetTask(c.Task)

	err := s.InitLocalWorkspace(c.Task.LinkId)
	c.updateRecord(10)
	if err != nil {
		c.failHandling(&s)
		return err
	}
	err = s.InitRemoteVersion(c.Task.LinkId)
	c.updateRecord(10)
	if err != nil {
		c.failHandling(&s)
		return err
	}
	err = s.PreDeploy(c.Task.LinkId)
	c.updateRecord(20)
	if err != nil {
		c.failHandling(&s)
		return err
	}
	if c.Project.RepoType == "git" {
		g := components.BaseGit{}
		g.SetBaseComponents(s)
		err = g.UpdateToVersion()
		c.updateRecord(30)
		if err != nil {
			c.failHandling(&s)
			return err
		}
	} else if c.Project.RepoType == "file" || c.Project.RepoType == "jenkins" {
		f := components.BaseFile{}
		f.SetBaseComponents(s)
		err = f.UpdateToVersion()
		c.updateRecord(30)
		if err != nil {
			c.failHandling(&s)
			return err
		}
	}

	err = s.PostDeploy(c.Task.LinkId)
	c.updateRecord(40)
	if err != nil {
		c.failHandling(&s)
		return err
	}
	err = s.CopyFiles()
	c.updateRecord(50)
	if err != nil {
		c.failHandling(&s)
		return err
	}
	err = s.UpdateRemoteServers(c.Task.LinkId)
	c.updateRecord(60)
	if err != nil {
		c.failHandling(&s)
		return err
	}
	//这里实际发布已完成 (后置本地脚本任务,)
	err = s.LastDeploy(c.Task.LinkId)
	if err != nil {
		c.failHandling(&s)
		return err
	}

	err = s.CleanUpLocal(c.Task.LinkId)
	c.updateRecord(100)
	if err != nil {
		return err
	}

	err = c.changeReleaseData()
	if err != nil {
		return err
	}

	return nil

}

func (c *releaseJob) changeReleaseData() error {
	//对于回滚的任务不记录线上版本
	if c.Task.Action == 0 {
		c.Task.ExLinkId = c.Project.Version
	}
	//判断是否为第一次任务，或者为回滚任务
	if c.Project.Version == "" || c.Task.Action == 1 {
		c.Task.EnableRollback = 0
	}
	c.Task.Status = 3
	c.Task.IsRun = 0
	c.Task.UpdatedAt = time.Now()
	err := models.UpdateTaskById(c.Task)
	if err != nil {
		return err
	}
	err = c.enableRollBack()
	if err != nil {
		return err
	}
	// 记录当前线上版本（软链）回滚则是回滚的版本，上线为新版本
	c.Project.Version = c.Task.LinkId
	err = models.UpdateProjectById(c.Project)
	if err != nil {
		return err
	}
	return nil
}

func (c *releaseJob) enableRollBack() error {
	ids, err := db.Values("SELECT id FROM task WHERE `status`=3 and project_id = ? and  `enable_rollback`=1 ORDER BY id DESC LIMIT ?", c.Task.ProjectId, c.Project.KeepVersionNum)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}

	// 保留的版本 id 集合，改用占位符传入避免拼接 SQL
	keepIds := []interface{}{}
	for _, id := range ids {
		keepIds = append(keepIds, common.GetString(id["id"]))
	}

	versionsRes, err := db.Values("SELECT link_id FROM task WHERE `enable_rollback`=1 and `id` not in (?) and  project_id = ? ", keepIds, c.Task.ProjectId)
	if err == nil && len(versionsRes) > 0 {
		var versions []string
		for _, version := range versionsRes {
			versions = append(versions, common.GetString(version["link_id"]))
		}
		//这里查找需要设置不可回滚的版本 进行清除操作
		s := components.BaseComponents{}
		s.SetProject(c.Project)
		s.SetTask(c.Task)
		s.CleanUpReleasesVersion(versions)
	}
	_, err = db.Exec("UPDATE `task` SET `enable_rollback`='0' WHERE`id` not in (?) and  project_id = ? and  `enable_rollback`=1 ", keepIds, c.Task.ProjectId)
	return err
}

// 上线失败处理
func (c *releaseJob) failHandling(co *components.BaseComponents) {
	//修改状态
	c.Task.Status = 4
	c.Task.IsRun = 0
	c.Task.UpdatedAt = time.Now()
	models.UpdateTaskById(c.Task)
	//清理本地版本库
	co.CleanUpLocal(c.Task.LinkId)
}
