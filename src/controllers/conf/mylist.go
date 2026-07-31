package confcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/common"
	"github.com/linclin/gopub/src/library/db"
)

func MyList(c echo.Context) error {
	ctx := controllers.New(c)
	// 原实现直接访问 ctx.User.Role，未登录时会 panic，这里与其他接口保持一致先行拦截
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}
	page, _ := ctx.GetInt("page", 0)
	start := 0
	length, _ := ctx.GetInt("length", 200000)
	if page > 0 {
		start = (page - 1) * length
	}
	selectInfo := ctx.GetString("select_info")
	// 原实现把 select_info 直接拼进 SQL，这里改为参数化查询避免注入
	where := ""
	args := []interface{}{}
	if selectInfo != "" {
		where += " and (`name` LIKE ?) "
		args = append(args, "%"+selectInfo+"%")
	}
	if ctx.User.Role == 10 {
		where += "and  `level`= 2  "
	} else if ctx.User.Role == 20 {
		where += "and  id in (SELECT project_id FROM `group` WHERE `group`.user_id= ? )  "
		args = append(args, ctx.User.Id)
	}

	listArgs := append(append([]interface{}{}, args...), start, length)
	projects, _ := db.Values("SELECT *, (SELECT realname FROM `user` WHERE `user`.id=project.user_id LIMIT 1) as realname,(SELECT realname FROM `user` WHERE `user`.id=project.user_lock LIMIT 1) as lockuser FROM `project`  WHERE 1=1 "+where+" ORDER BY id LIMIT ?,?", listArgs...)

	total := 0
	count, _ := db.Values("SELECT count(id) FROM `project` WHERE 1=1 "+where, args...)
	if len(count) > 0 {
		total = common.GetInt(count[0]["count(id)"])
	}
	return ctx.SetJson(0, map[string]interface{}{"total": total, "currentPage": page, "table_data": projects}, "")
}
