package othercontrollers

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/common"
	"github.com/linclin/gopub/src/library/db"
	"github.com/linclin/gopub/src/library/logger"
)

// 这里是查询每天 每周 每月 未进入预发布的项目
func NoAuto(c echo.Context) error {
	ctx := controllers.New(c)
	taskType := ctx.GetString("taskType")
	logger.Info(taskType)
	sql := "SELECT project.id ,project.name  FROM `task` LEFT JOIN project ON task.project_id=project.id WHERE  project.level=2 %s group BY project.id"
	var proIds []db.Params
	var proIds1 []db.Params
	if taskType == "day" {
		proIds, _ = db.Values(fmt.Sprintf(sql, " and TO_DAYS(now()) - TO_DAYS(task.updated_at) = 0 "))
		proIds, _ = db.Values(fmt.Sprintf(sql, " and TO_DAYS(now()) - TO_DAYS(task.updated_at) = 0  and task.user_id=1"))
	} else if taskType == "week" {
		proIds, _ = db.Values(fmt.Sprintf(sql, " and YEARWEEK(date_format(task.updated_at,'%Y-%m-%d')) = YEARWEEK(now()) "))
		proIds1, _ = db.Values(fmt.Sprintf(sql, " and YEARWEEK(date_format(task.updated_at,'%Y-%m-%d')) = YEARWEEK(now()) and task.user_id=1"))
	} else {
		proIds, _ = db.Values(fmt.Sprintf(sql, " and date_format(task.updated_at,'%Y-%m')=date_format(now(),'%Y-%m') "))
		proIds1, _ = db.Values(fmt.Sprintf(sql, " and date_format(task.updated_at,'%Y-%m')=date_format(now(),'%Y-%m')  and task.user_id=1"))
	}
	res := []db.Params{}
	for _, proId := range proIds {
		id := common.GetInt(proId["id"])
		isIn := false
		for _, proId1 := range proIds1 {
			id1 := common.GetInt(proId1["id"])
			if id == id1 {
				isIn = true
			}
		}
		if !isIn {
			res = append(res, proId)
		}
	}
	return ctx.SetJson(0, res, "")
}
