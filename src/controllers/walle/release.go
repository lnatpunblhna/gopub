package wallecontrollers

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
)

// releaseJob 承载一次上线/回滚任务所需的数据。
//
// 上线流程在独立 goroutine 中执行，而 Echo 会复用 echo.Context 对象，
// 请求结束后继续持有它会读到其他请求的数据，因此这里只保留 Task/Project，
// 不引用请求上下文。
type releaseJob struct {
	Task    *models.Task
	Project *models.Project
	// Attempt 是这次发布在该上线单下的批次号，用于把多次发布的日志分开保存
	Attempt int
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
	// 这里原先是 DELETE 掉该任务的全部记录再重新发布，
	// 上一次为什么失败连同日志一起没了；改为开一个新批次，历史留着可查
	job := &releaseJob{
		Task:    ctx.Task,
		Project: project,
		Attempt: models.NextAttempt(int64(ctx.Task.Id)),
	}
	if job.Task.Action == 0 {
		//生成版本号
		job.makeVersion()
		go func() {
			if err := job.releaseHandling(); err != nil {
				job.logErr(err)
			}
		}()
	} else {
		// 回滚原先是裸 go 调用，返回的 error 直接被丢掉，
		// 失败时既没有日志也没有记录，只能靠猜
		go func() {
			if err := job.rollBackHandling(); err != nil {
				job.logErr(err)
			}
		}()
	}
	// 把本次批次号回给前端：日志页据此直接跟踪这一轮，
	// 不用再去猜「最新批次」，避开记录还没落库时的竞态
	return ctx.SetJson(0, map[string]interface{}{"attempt": job.Attempt}, "")
}

// logErr 把流程级错误同时写进日志与 task_err_log
func (c *releaseJob) logErr(err error) {
	logger.Error("上线任务失败 taskId=", c.Task.Id, " ", err)
	if _, e := models.AddTaskErrLog(&models.TaskErrLog{
		ErrInfo: err.Error(),
		TaskId:  c.Task.Id,
	}); e != nil {
		logger.Error("写入 task_err_log 失败:", e)
	}
}

// newComponents 构造带好归属信息的执行组件
func (c *releaseJob) newComponents() components.BaseComponents {
	s := components.BaseComponents{}
	s.SetProject(c.Project)
	s.SetTask(c.Task)
	s.SetAttempt(c.Attempt)
	if c.Task.Action == 0 {
		s.SetScope(models.RecordScopeRelease)
	} else {
		s.SetScope(models.RecordScopeRollback)
	}
	return s
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
	s := c.newComponents()
	err := s.UpdateRemoteServers(c.Task.LinkId)
	if err != nil {
		c.failHandling(&s, "回滚目标机版本", err)
		return err
	}
	err = c.changeReleaseData()
	if err != nil {
		c.failHandling(&s, "更新回滚结果", err)
		return err
	}
	s.AddFinalRecord(true, "", nil)
	return nil
}

// 普通上线任务
//
// 两处与旧实现的差别：
//  1. 先判错再推进。旧实现是 updateRecord 在 if err != nil 之前调用，
//     失败的那一步也会被标成已完成，步骤条照样往前走，看不出卡在哪。
//  2. 阶段在记录写入时就打好（SetStage），不再事后 UPDATE 回填。
func (c *releaseJob) releaseHandling() error {
	s := c.newComponents()

	steps := []struct {
		name  string
		stage uint // 与前端步骤条的 6 步对应：10=1 ... 60=6
		run   func() error
	}{
		{"初始化本地工作区", 10, func() error { return s.InitLocalWorkspace(c.Task.LinkId) }},
		{"初始化目标机版本库", 10, func() error { return s.InitRemoteVersion(c.Task.LinkId) }},
		{"pre-deploy 任务", 20, func() error { return s.PreDeploy(c.Task.LinkId) }},
		{"检出代码", 30, func() error { return c.checkout(s) }},
		{"post-deploy 任务", 40, func() error { return s.PostDeploy(c.Task.LinkId) }},
		{"同步文件至目标机", 50, func() error { return s.CopyFiles() }},
		{"更新目标机版本", 60, func() error { return s.UpdateRemoteServers(c.Task.LinkId) }},
		//这里实际发布已完成 (后置本地脚本任务,)
		{"last-deploy 任务", 60, func() error { return s.LastDeploy(c.Task.LinkId) }},
	}

	for _, step := range steps {
		s.SetStage(step.stage)
		if err := step.run(); err != nil {
			c.failHandling(&s, step.name, err)
			return err
		}
	}

	// 走到这里目标机已经切到新版本，清理宿主机临时目录失败不该判成发布失败。
	// 失败详情在上面那条 record 里已经有了，这里只补一条日志。
	// （旧实现在这里直接 return，导致 task.is_run 一直是 1，任务永远卡在"正在上线中"）
	if err := s.CleanUpLocal(c.Task.LinkId); err != nil {
		logger.Error("清理本地工作区失败 taskId=", c.Task.Id, " ", err)
	}

	if err := c.changeReleaseData(); err != nil {
		c.failHandling(&s, "更新上线结果", err)
		return err
	}

	s.AddFinalRecord(true, "", nil)
	return nil
}

// checkout 按仓库类型检出代码
func (c *releaseJob) checkout(s components.BaseComponents) error {
	switch c.Project.RepoType {
	case "git":
		g := components.BaseGit{}
		g.SetBaseComponents(s)
		return g.UpdateToVersion()
	case "file", "jenkins":
		f := components.BaseFile{}
		f.SetBaseComponents(s)
		return f.UpdateToVersion()
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
		s := c.newComponents()
		if err := s.CleanUpReleasesVersion(versions); err != nil {
			logger.Error("清理历史版本失败 taskId=", c.Task.Id, " ", err)
		}
	}
	_, err = db.Exec("UPDATE `task` SET `enable_rollback`='0' WHERE`id` not in (?) and  project_id = ? and  `enable_rollback`=1 ", keepIds, c.Task.ProjectId)
	return err
}

// 上线失败处理
//
// 除了改任务状态，还要留下「哪一步、为什么失败」：
// 旧实现只改状态，前端既看不到原因，也不知道流程已经终止，会一直空转轮询。
func (c *releaseJob) failHandling(co *components.BaseComponents, stage string, failErr error) {
	//修改状态
	c.Task.Status = 4
	c.Task.IsRun = 0
	c.Task.UpdatedAt = time.Now()
	if err := models.UpdateTaskById(c.Task); err != nil {
		logger.Error("更新任务状态失败 taskId=", c.Task.Id, " ", err)
	}
	//清理本地版本库
	if err := co.CleanUpLocal(c.Task.LinkId); err != nil {
		logger.Error("清理本地工作区失败 taskId=", c.Task.Id, " ", err)
	}
	// 终结记录要放在清理之后写，保证它是这次上线的最后一条
	co.AddFinalRecord(false, stage, failErr)
}
