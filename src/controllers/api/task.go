package apicontrollers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/models"
)

// oprations for Task

// TaskPost 创建 Task
// @router / [post]
func TaskPost(c echo.Context) error {
	ctx := controllers.New(c)
	var v models.Task
	if err := json.Unmarshal(ctx.RequestBody(), &v); err != nil {
		return ctx.Json(err.Error())
	}
	if _, err := models.AddTask(&v); err != nil {
		return ctx.Json(err.Error())
	}
	return ctx.JsonStatus(http.StatusCreated, v)
}

// TaskGetOne 按 id 获取 Task
// @router /:id [get]
func TaskGetOne(c echo.Context) error {
	ctx := controllers.New(c)
	id, _ := strconv.Atoi(c.Param("id"))
	v, err := models.GetTaskById(id)
	if err != nil {
		return ctx.Json(err.Error())
	}
	return ctx.Json(v)
}

// TaskGetAllAndProName 按项目名与时间区间查询已上线任务
// @router / [get]
func TaskGetAllAndProName(c echo.Context) error {
	ctx := controllers.New(c)
	startDate := ctx.GetString("startTime")
	endDate := ctx.GetString("endTime")
	projectName := ctx.GetString("project_name")
	if startDate != "" {
		if _, err := time.Parse("2006-01-02 15:04:05", startDate); err != nil {
			return ctx.Json(map[string]string{"errcode": "103", "errmsg": "start时间格式不合法"})
		}
	}
	if endDate != "" {
		if _, err := time.Parse("2006-01-02 15:04:05", endDate); err != nil {
			return ctx.Json(map[string]string{"errcode": "104", "errmsg": "end时间格式不合法"})
		}
	}
	l, err := models.GetAllTaskAndPro(projectName, startDate, endDate)
	if err != nil {
		return ctx.Json(err.Error())
	}
	return ctx.Json(l)
}

// TaskGetAll 通用条件查询 Task。
//
// 原 beego 版本中 GET /v1/task 已被 TaskGetAllAndProName 占用
// （URLMapping 把 "GetAll" 映射到了 GetAllAndProName），该接口实际不可达，
// 迁移后同样不注册路由，保留实现以备后用。
func TaskGetAll(c echo.Context) error {
	ctx := controllers.New(c)
	var fields []string
	var sortby []string
	var order []string
	var query map[string]string = make(map[string]string)
	var limit int64 = 10
	var offset int64 = 0

	// fields: col1,col2,entity.col3
	if v := ctx.GetString("fields"); v != "" {
		fields = strings.Split(v, ",")
	}
	// limit: 10 (default is 10)
	if v, err := ctx.GetInt64("limit"); err == nil {
		limit = v
	}
	// offset: 0 (default is 0)
	if v, err := ctx.GetInt64("offset"); err == nil {
		offset = v
	}
	// sortby: col1,col2
	if v := ctx.GetString("sortby"); v != "" {
		sortby = strings.Split(v, ",")
	}
	// order: desc,asc
	if v := ctx.GetString("order"); v != "" {
		order = strings.Split(v, ",")
	}
	// query: k:v,k:v
	if v := ctx.GetString("query"); v != "" {
		for _, cond := range strings.Split(v, ",") {
			kv := strings.Split(cond, ":")
			if len(kv) != 2 {
				return ctx.Json(errors.New("Error: invalid query key/value pair").Error())
			}
			k, v := kv[0], kv[1]
			query[k] = v
		}
	}

	l, err := models.GetAllTask(query, fields, sortby, order, offset, limit)
	if err != nil {
		return ctx.Json(err.Error())
	}
	return ctx.Json(l)
}

// TaskPut 更新 Task
// @router /:id [put]
func TaskPut(c echo.Context) error {
	ctx := controllers.New(c)
	id, _ := strconv.Atoi(c.Param("id"))
	v := models.Task{Id: id}
	if err := json.Unmarshal(ctx.RequestBody(), &v); err != nil {
		return ctx.Json(err.Error())
	}
	if err := models.UpdateTaskById(&v); err != nil {
		return ctx.Json(err.Error())
	}
	return ctx.Json("OK")
}

// TaskDelete 删除 Task
// @router /:id [delete]
func TaskDelete(c echo.Context) error {
	ctx := controllers.New(c)
	id, _ := strconv.Atoi(c.Param("id"))
	if err := models.DeleteTask(id); err != nil {
		return ctx.Json(err.Error())
	}
	return ctx.Json("OK")
}
