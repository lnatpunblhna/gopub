package recordcontrollers

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/models"
)

// Log 下载某条记录的完整输出。
//
// 入库的 memo 是截断过的（memo 列有长度上限，写超了整条更新都会失败），
// 完整内容落在 logs/task_log 下，需要细查时从这里取。
func Log(c echo.Context) error {
	ctx := controllers.New(c)
	recordId, _ := ctx.GetInt("recordId", 0)
	if recordId <= 0 {
		return ctx.SetJson(1, nil, "参数错误")
	}
	re, err := models.GetRecordById(recordId)
	if err != nil || re == nil {
		return ctx.SetJson(1, nil, "记录不存在")
	}
	if !canReadRecord(ctx.User, re) {
		return ctx.SetJson(1, nil, "无权限查看该记录")
	}
	full, ok := components.TaskLogPath(components.LogFileOfMemo(re.Memo))
	if !ok {
		return ctx.SetJson(1, nil, "该记录没有留存完整日志")
	}
	return ctx.Attachment(full, fmt.Sprintf("record-%d.log", re.Id))
}

// canReadRecord 判断能否查看某条记录的完整日志。
// 有上线单的按上线单所属项目判断，没有的（检测 / 刷新等）落到项目权限或本人。
func canReadRecord(user *models.User, re *models.Record) bool {
	if user == nil || user.Id == 0 {
		return false
	}
	if re.TaskId > 0 {
		task, err := models.GetTaskById(int(re.TaskId))
		if err != nil {
			return false
		}
		return controllers.CanAccessTask(user, task)
	}
	if re.ProjectId > 0 {
		return controllers.CanAccessProjectId(user, re.ProjectId)
	}
	return re.UserId == uint(user.Id)
}
