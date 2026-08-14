package taskcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/db"
)

// List 返回全部上线单，只给管理员用（前端「全部上线单」页）。
func List(c echo.Context) error {
	return listTasks(controllers.New(c), 0)
}

// MyList 只返回当前登录用户自己创建的上线单。
//
// 用户 ID 一律取自登录态。原先前端把自己的 user.Id 当 my 参数传上来、后端照单
// 全收（listTasks 的 myUserId），把参数改成别人的 ID 就能看别人的上线单。
func MyList(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}
	return listTasks(ctx, ctx.User.Id)
}

// listTasks 是两个列表接口共用的查询。userId 非 0 时只返回该用户创建的上线单。
func listTasks(ctx *controllers.Ctx, userId int) error {
	page, _ := ctx.GetInt("page", 0)
	start := 0
	length, _ := ctx.GetInt("length", 15)
	if page > 0 {
		start = (page - 1) * length
	}
	// 原实现把 select_info 直接拼进 SQL，这里改为参数化查询避免注入
	selectInfo := ctx.GetString("select_info")
	where := ""
	args := []interface{}{}
	if selectInfo != "" {
		where += "  and( project.`name` LIKE ? or `user`.realname LIKE ?  or task.title LIKE ?  )"
		like := "%" + selectInfo + "%"
		args = append(args, like, like, like)
	}
	if userId != 0 {
		where += "  and task.user_id= ? "
		args = append(args, userId)
	}

	listArgs := append(append([]interface{}{}, args...), start, length)
	projects, _ := db.Values("SELECT task.id,project.name,project.name,project.level,`user`.realname,task.title,task.action,task.link_id,task.is_run,task.enable_rollback,task.updated_at,task.branch,task.commit_id,task.pms_uwork_id,task.pms_batch_id,task.`status` FROM `task` LEFT JOIN project on task.project_id=project.id   LEFT JOIN `user` on task.user_id=user.id where 1=1 "+where+" order by task.id DESC  LIMIT ? ,?", listArgs...)

	total := 0
	count, _ := db.Values("SELECT count(task.id) FROM `task` LEFT JOIN project on task.project_id=project.id   LEFT JOIN `user` on task.user_id=user.id where 1=1 "+where, args...)
	if len(count) > 0 {
		total = common.GetInt(count[0]["count(task.id)"])
	}
	for _, project := range projects {
		project["status"] = GetTaskStatus(common.GetInt(project["status"]))
		if common.GetInt(project["is_run"]) != 0 && common.GetString(project["status"]) != "上线完成" {
			project["status"] = "上线中"
		}
		if common.GetInt(project["level"]) == 3 {
			project["name"] = common.GetString(project["name"]) + "-线上环境"
		}
		if common.GetInt(project["level"]) == 2 {
			project["name"] = common.GetString(project["name"]) + "-预发布环境"
		}
	}
	return ctx.SetJson(0, map[string]interface{}{"total": total, "currentPage": page, "table_data": projects}, "")
}

func GetTaskStatus(status int) string {
	switch status {
	case 0:
		return "新建提交"
	case 1:
		return "新建提交"
	case 2:
		return "审核拒绝"
	case 3:
		return "上线完成"
	case 4:
		return "上线失败"
	}
	return ""
}
