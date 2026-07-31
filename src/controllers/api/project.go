package apicontrollers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/models"
)

// oprations for Project
//
// 注意：原 beego 版本的 router.go 只把 /v1/token 与 /v1/task 挂进了命名空间，
// 并未注册 /v1/project，因此本文件的接口原本就不可达。
// 迁移后保持一致，不注册路由，仅保留实现。

// ProjectPost 创建 Project
// @router / [post]
func ProjectPost(c echo.Context) error {
	ctx := controllers.New(c)
	var v models.Project
	if err := json.Unmarshal(ctx.RequestBody(), &v); err != nil {
		return ctx.Json(err.Error())
	}
	if _, err := models.AddProject(&v); err != nil {
		return ctx.Json(err.Error())
	}
	return ctx.JsonStatus(http.StatusCreated, v)
}

// ProjectGetOne 按 id 获取 Project
// @router /:id [get]
func ProjectGetOne(c echo.Context) error {
	ctx := controllers.New(c)
	id, _ := strconv.Atoi(c.Param("id"))
	v, err := models.GetProjectById(id)
	if err != nil {
		return ctx.Json(err.Error())
	}
	return ctx.Json(v)
}

// ProjectGetAll 通用条件查询 Project
// @router / [get]
func ProjectGetAll(c echo.Context) error {
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

	l, err := models.GetAllProject(query, fields, sortby, order, offset, limit)
	if err != nil {
		return ctx.Json(err.Error())
	}
	return ctx.Json(l)
}

// ProjectPut 更新 Project
// @router /:id [put]
func ProjectPut(c echo.Context) error {
	ctx := controllers.New(c)
	id, _ := strconv.Atoi(c.Param("id"))
	v := models.Project{Id: id}
	if err := json.Unmarshal(ctx.RequestBody(), &v); err != nil {
		return ctx.Json(err.Error())
	}
	if err := models.UpdateProjectById(&v); err != nil {
		return ctx.Json(err.Error())
	}
	return ctx.Json("OK")
}

// ProjectDelete 删除 Project
// @router /:id [delete]
func ProjectDelete(c echo.Context) error {
	ctx := controllers.New(c)
	id, _ := strconv.Atoi(c.Param("id"))
	if err := models.DeleteProject(id); err != nil {
		return ctx.Json(err.Error())
	}
	return ctx.Json("OK")
}
