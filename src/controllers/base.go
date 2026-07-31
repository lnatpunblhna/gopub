// Package controllers 提供 Echo handler 的公共请求上下文。
//
// 取值语义精确对齐原 beego 控制器，避免迁移后前端行为变化：
//   - GetString/GetInt 依次查路由参数、URL query、表单（对应 beego 的 Ctx.Input.Query）
//   - GetString 取到空串时返回默认值；GetInt 值为空且给了默认值时返回默认值
//   - RequestBody 可重复读取（对应原 app.conf 中的 CopyRequestBody = true）
package controllers

import (
	"bytes"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/library/common"
	"github.com/linclin/gopub/src/library/db"
	"github.com/linclin/gopub/src/library/logger"
	"github.com/linclin/gopub/src/models"
)

// Ctx 是各 handler 共用的请求上下文，取代原先的 BaseController。
type Ctx struct {
	echo.Context
	Project *models.Project
	Task    *models.Task
	User    *models.User

	body []byte
}

// New 构造请求上下文，并完成原 BaseController.Prepare 所做的解析：
// 按 taskId / projectId 预加载对应记录，按 Authorization 头识别当前用户。
func New(c echo.Context) *Ctx {
	ctx := &Ctx{Context: c}
	ctx.cacheBody()

	// 与原 Prepare 一致：解析过程中的异常只记录日志，不中断请求
	defer func() {
		if panicErr := recover(); panicErr != nil {
			buf := make([]byte, 1024)
			n := runtime.Stack(buf, false)
			logger.Error("控制器错误:", panicErr, string(buf[0:n]))
		}
	}()

	if taskId := ctx.GetString("taskId"); taskId != "" {
		ctx.Task, _ = models.GetTaskById(common.GetInt(taskId))
	}
	if projectId := ctx.GetString("projectId"); projectId != "" {
		ctx.Project, _ = models.GetProjectById(common.GetInt(projectId))
	}
	ctx.User = ctx.userByToken()
	return ctx
}

// userByToken 解析 `Authorization: TOKEN xxx` 头并查出对应用户
func (c *Ctx) userByToken() *models.User {
	ah := c.Request().Header.Get("Authorization")
	if len(ah) <= 5 || strings.ToUpper(ah[0:5]) != "TOKEN" {
		return nil
	}
	token := ah[6:]
	if token == "" {
		return nil
	}
	var users []models.User
	n, err := db.QueryRows(&users, "SELECT * FROM `user` WHERE auth_key= ?", token)
	if n > 0 && err == nil {
		return &users[0]
	}
	return nil
}

// cacheBody 读出请求体并复位，使其可被重复读取
func (c *Ctx) cacheBody() {
	req := c.Request()
	if req.Body == nil {
		return
	}
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return
	}
	c.body = data
	req.Body = io.NopCloser(bytes.NewReader(data))
}

// RequestBody 返回请求体，等价于 beego 的 c.Ctx.Input.RequestBody
func (c *Ctx) RequestBody() []byte {
	return c.body
}

// query 复刻 beego Ctx.Input.Query：路由参数优先，其次 URL query，最后表单
func (c *Ctx) query(key string) string {
	if v := c.Param(key); v != "" {
		return v
	}
	if v := c.QueryParam(key); v != "" {
		return v
	}
	return c.FormValue(key)
}

// GetString 等价于 beego 的 c.GetString
func (c *Ctx) GetString(key string, def ...string) string {
	if v := c.query(key); v != "" {
		return v
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

// GetInt 等价于 beego 的 c.GetInt
func (c *Ctx) GetInt(key string, def ...int) (int, error) {
	v := c.query(key)
	if v == "" && len(def) > 0 {
		return def[0], nil
	}
	return strconv.Atoi(v)
}

// GetInt64 等价于 beego 的 c.GetInt64
func (c *Ctx) GetInt64(key string, def ...int64) (int64, error) {
	v := c.query(key)
	if v == "" && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseInt(v, 10, 64)
}

// GetBool 等价于 beego 的 c.GetBool
func (c *Ctx) GetBool(key string, def ...bool) (bool, error) {
	v := c.query(key)
	if v == "" && len(def) > 0 {
		return def[0], nil
	}
	return strconv.ParseBool(v)
}

// IP 返回客户端 IP，等价于 beego 的 c.Ctx.Input.IP()
func (c *Ctx) IP() string {
	return c.RealIP()
}

// Header 返回请求头，等价于 beego 的 c.Ctx.Input.Header(key)
func (c *Ctx) Header(key string) string {
	return c.Request().Header.Get(key)
}

// Site 返回站点地址，等价于 beego 的 c.Ctx.Input.Site()
func (c *Ctx) Site() string {
	return c.Scheme() + "://" + strings.Split(c.Request().Host, ":")[0]
}

// IsSecure 判断是否为 HTTPS 请求，等价于 beego 的 c.Ctx.Input.IsSecure()
func (c *Ctx) IsSecure() bool {
	return c.Scheme() == "https"
}

// SetJson 输出统一结构的响应体，等价于原 BaseController.SetJson
func (c *Ctx) SetJson(code int, data interface{}, msg string) error {
	if code == 0 && msg == "" {
		msg = "sucess"
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"code": code, "msg": msg, "data": data})
}

// Json 直接输出数据，等价于 beego 的 c.Data["json"] = x 后 c.ServeJSON()
func (c *Ctx) Json(data interface{}) error {
	return c.JSON(http.StatusOK, data)
}

// JsonStatus 以指定状态码输出数据，等价于 beego 的 Ctx.Output.SetStatus 后 ServeJSON
func (c *Ctx) JsonStatus(status int, data interface{}) error {
	return c.JSON(status, data)
}
