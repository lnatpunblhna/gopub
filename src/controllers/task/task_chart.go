package taskcontrollers

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/cache"
	"github.com/linclin/gopub/src/library/common"
	"github.com/linclin/gopub/src/library/components"
	"github.com/linclin/gopub/src/library/db"
	"github.com/linclin/gopub/src/library/logger"
	"github.com/linclin/gopub/src/models"
)

var bm = cache.New()

func TaskChart(c echo.Context) error {
	ctx := controllers.New(c)
	taskType := ctx.GetString("taskType")

	// 按环境级别统计的三种时间维度
	levelSQL := map[string]string{
		"day":   "SELECT project.`level`,count(task.id) as task_count  FROM `task` LEFT JOIN project ON task.project_id = project.id WHERE TO_DAYS(now()) - TO_DAYS(task.updated_at) = 0 GROUP BY project. LEVEL",
		"week":  "SELECT project.`level`,count(task.id) as task_count  FROM `task` LEFT JOIN project ON task.project_id = project.id WHERE YEARWEEK(date_format(task.updated_at,'%Y-%m-%d')) = YEARWEEK(now()) GROUP BY project. LEVEL",
		"month": "SELECT project.`level`,count(task.id) as task_count  FROM `task` LEFT JOIN project ON task.project_id = project.id WHERE date_format(task.updated_at,'%Y-%m')=date_format(now(),'%Y-%m') GROUP BY project. LEVEL",
	}
	// 按项目统计的三种时间维度
	proSQL := map[string]string{
		"dayBypro":   "SELECT project.`name`,count(task.id) as task_count,project.`level` FROM `task` LEFT JOIN project ON task.project_id = project.id WHERE TO_DAYS(now()) - TO_DAYS(task.updated_at) = 0 and task.status=3 GROUP BY project.id",
		"weekBypro":  "SELECT project.`name`,count(task.id) as task_count,project.`level` FROM `task` LEFT JOIN project ON task.project_id = project.id WHERE YEARWEEK(date_format(task.updated_at,'%Y-%m-%d')) = YEARWEEK(now()) and task.status=3 GROUP BY project.id",
		"monthBypro": "SELECT project.`name`,count(task.id) as task_count,project.`level` FROM `task` LEFT JOIN project ON task.project_id = project.id WHERE date_format(task.updated_at,'%Y-%m')=date_format(now(),'%Y-%m') and task.status=3 GROUP BY project.id",
	}

	if sql, ok := levelSQL[taskType]; ok {
		count, _ := db.Values(sql)
		for _, row := range count {
			row["name"] = GetProjectLevel(common.GetInt(row["level"]))
		}
		if taskType == "month" {
			logger.Info(count)
		}
		return ctx.SetJson(0, count, "")
	}

	if sql, ok := proSQL[taskType]; ok {
		count, _ := db.Values(sql)
		for _, row := range count {
			row["name"] = common.GetString(row["name"]) + "-" + GetProjectLevel(common.GetInt(row["level"]))
		}
		return ctx.SetJson(0, count, "")
	}

	if taskType == "total" {
		totalJson := map[string]interface{}{}
		for key, sql := range map[string]string{
			"totalmen":        "SELECT count(id) as `totalmen` FROM `user`",
			"totalproject":    "SELECT count(DISTINCT name) as `totalproject` from `project`",
			"totalpub":        "SELECT count(id) as `totalpub` from `task`",
			"totalpubsuccess": "SELECT count(id) as `totalpubsuccess` from `task`where status = 3",
		} {
			rows, err := db.Values(sql)
			if err == nil && len(rows) > 0 {
				totalJson[key] = common.GetInt(rows[0][key])
			}
		}
		if !bm.IsExist("hostsum") {
			totalJson["hostsum"] = GetHostNum()
		} else {
			totalJson["hostsum"] = bm.Get("hostsum")
		}
		return ctx.SetJson(0, totalJson, "")
	}
	return ctx.SetJson(1, nil, "未传参数")
}

func GetProjectLevel(level int) string {
	switch level {
	case 1:
		return "测试环境"
	case 2:
		return "预发布环境"
	case 3:
		return "生产环境"
	}
	return "删除项目"
}

func GetHostNum() int {
	var projects []models.Project
	i, err := db.QueryRows(&projects, "SELECT * FROM `project`")
	finalres := []string{}
	if i > 0 && err == nil {
		for _, project := range projects {
			s := components.BaseComponents{}
			s.SetProject(&project)
			ips := s.GetHostIps()
			for _, ip := range ips {
				if !common.InList(ip, finalres) {
					finalres = append(finalres, ip)
				}
			}
		}
	}
	bm.Put("hostsum", len(finalres), 1*time.Hour)
	return len(finalres)
}
