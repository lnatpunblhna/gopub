package confcontrollers

import (
	"strings"

	"github.com/astaxie/beego/orm"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/common"
)

type ListController struct {
	controllers.BaseController
}

func (c *ListController) Get() {
	page, _ := c.GetInt("page", 0)
	start := 0
	length, _ := c.GetInt("length", 200000)
	if page > 0 {
		start = (page - 1) * length
	}
	selectInfo := c.GetString("select_info")
	where := ""
	args := []interface{}{}
	if selectInfo != "" {
		where = " AND project.`name` LIKE ?"
		args = append(args, "%"+selectInfo+"%")
	}
	var projects []orm.Params
	o := orm.NewOrm()

	listSQL := "SELECT project.id, project.name, project.level, project.repo_type, project.keep_version_num, project.p2p, COALESCE(NULLIF(`user`.realname, ''), `user`.username, '') AS realname, DATE_FORMAT(project.created_at, '%Y-%m-%d %H:%i:%s') AS created_at, DATE_FORMAT(project.updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at FROM `project` LEFT JOIN `user` ON `user`.id = project.user_id WHERE 1=1" + where + " ORDER BY project.id LIMIT ?,?"
	listArgs := append(args, start, length)
	o.Raw(listSQL, listArgs...).Values(&projects)
	var count []orm.Params
	total := 0
	countSQL := "SELECT count(id) AS total FROM `project` WHERE 1=1" + strings.Replace(where, "project.`name`", "`name`", 1)
	o.Raw(countSQL, args...).Values(&count)
	if len(count) > 0 {
		total = common.GetInt(count[0]["total"])
	}
	c.SetJson(0, map[string]interface{}{"total": total, "currentPage": page, "table_data": projects}, "")

	return

}
