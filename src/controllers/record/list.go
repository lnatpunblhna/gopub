package recordcontrollers

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
)

// maxRecordPage 单次返回的记录条数上限。
// 配合 lastId 增量拉取使用：前端每 2 秒只取新增的那几条，
// 不再像以前那样把整个任务的日志（含 memo 大字段）全量重取一遍。
const maxRecordPage = 200

const recordColumns = "`id`,`user_id`,`task_id`,`project_id`,`scope`,`attempt`,`status`,`action`,`command`,`duration`,`memo`,`created_at`"

// maskedMemo 是脱敏后写回给未登录请求的 memo，结构与正常 memo 一致，前端无需特判
const maskedMemo = `{"kind":"masked","success":true,"hosts":[]}`

// List 返回上线进度记录。
//
// taskId > 0：真实上线单，按 task_id 增量拉取，默认只看最近一次发布。
// taskId <= 0：检测 / 刷新 / gitpull 这类没有上线单的操作，前端用固定负数占位。
// 这类记录以前全堆在同一个 task_id 上，谁点的都能看到别人的输出；
// 现在按 scope + 当前登录用户过滤，各看各的。
//
// 这个接口在免登录白名单里（支撑对外的发布看板页），
// 未登录时会脱敏：只保留步骤名与成功 / 失败，不返回命令、路径和主机输出。
func List(c echo.Context) error {
	ctx := controllers.New(c)
	taskId, _ := ctx.GetInt64("taskId", 0)
	lastId, _ := ctx.GetInt64("lastId", 0)

	var (
		records []db.Params
		err     error
	)
	if taskId > 0 {
		attempt, _ := ctx.GetInt("attempt", 0)
		if attempt <= 0 && lastId == 0 {
			// 没指定批次且是首次拉取：只给最近一次发布的日志，
			// 历史批次由前端显式带 attempt 参数来取
			attempt = models.MaxAttempt(taskId)
		}
		if attempt > 0 {
			records, err = db.Values(
				"SELECT "+recordColumns+" FROM `record` WHERE `task_id`=? AND `attempt`=? AND `id`>? ORDER BY `id` ASC LIMIT ?",
				taskId, attempt, lastId, maxRecordPage)
		} else {
			records, err = db.Values(
				"SELECT "+recordColumns+" FROM `record` WHERE `task_id`=? AND `id`>? ORDER BY `id` ASC LIMIT ?",
				taskId, lastId, maxRecordPage)
		}
	} else {
		scope := models.ScopeByPseudoTaskId(taskId)
		var userId uint
		if ctx.User != nil {
			userId = uint(ctx.User.Id)
		}
		if scope == "" || userId == 0 {
			return ctx.SetJson(0, []db.Params{}, "")
		}
		since, _ := ctx.GetInt64("time", 0)
		records, err = db.Values(
			"SELECT "+recordColumns+" FROM `record` WHERE `scope`=? AND `user_id`=? AND `id`>? AND `created_at`>? ORDER BY `id` ASC LIMIT ?",
			scope, userId, lastId, since, maxRecordPage)
	}
	if err != nil {
		// 旧实现把错误吞掉直接返回空数组，页面上表现为「没有日志」，
		// 排查时完全分不清是真没有还是查库挂了
		logger.Error("查询上线记录失败 taskId=", taskId, " ", err)
		return ctx.SetJson(1, nil, "查询上线记录失败")
	}
	if records == nil {
		records = []db.Params{}
	}
	if ctx.User == nil {
		maskRecords(records)
	}
	return ctx.SetJson(0, records, "")
}

// Attempts 返回某个上线单的发布批次列表，供前端切换历史发布。
// 只有批次号、时间与成败，不含命令与主机信息，因此与 List 一样允许未登录访问。
func Attempts(c echo.Context) error {
	ctx := controllers.New(c)
	taskId, _ := ctx.GetInt64("taskId", 0)
	if taskId <= 0 {
		return ctx.SetJson(0, []db.Params{}, "")
	}
	rows, err := db.Values(
		"SELECT `attempt`,MIN(`created_at`) AS `started_at`,MIN(`status`) AS `min_status`,COUNT(*) AS `total` "+
			"FROM `record` WHERE `task_id`=? GROUP BY `attempt` ORDER BY `attempt` DESC LIMIT 20", taskId)
	if err != nil {
		logger.Error("查询发布批次失败 taskId=", taskId, " ", err)
		return ctx.SetJson(1, nil, "查询发布批次失败")
	}
	if rows == nil {
		rows = []db.Params{}
	}
	return ctx.SetJson(0, rows, "")
}

// maskRecords 对未登录请求脱敏：命令里带着部署路径，memo 里带着主机 IP 与完整输出，
// 这些不该对匿名访问者可见；步骤名与成功 / 失败足够让看板页面显示进度。
func maskRecords(records []db.Params) {
	for _, r := range records {
		action := common.GetInt(r["action"])
		// 终结记录的 command 只有「发布完成 / 发布失败：<步骤名>」，不含敏感信息，保留
		if action < actionFinished {
			r["command"] = stageLabel(action)
		}
		r["memo"] = maskedMemo
	}
}

// actionFinished 与 components 里终结记录的 action 取值保持一致
const actionFinished = 100

func stageLabel(action int) string {
	if action >= 10 {
		return fmt.Sprintf("步骤 %d", action/10)
	}
	return "执行中"
}
