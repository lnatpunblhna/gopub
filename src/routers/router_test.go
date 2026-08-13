package routers

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// 允许未登录访问的路由：登录入口、前端入口页，以及免登录看日志页用到的只读查询。
// 其余内部路由都必须被 RequireLogin / RequireAdmin 拦住。
var noAuthRoutes = map[string]bool{
	"POST /login":                  true,
	"GET /loginbydocke":            true,
	"GET /":                        true,
	"GET /api/get/task/list":       true,
	"GET /api/get/task/get":        true,
	"GET /api/get/conf/get":        true,
	"GET /api/get/record/list":     true,
	"GET /api/get/record/attempts": true,
	"GET /v1/token":                true, // 签发 token 的接口本身不能要求带 token
}

// wantRoutes 是预期的全部路由，用于确认改动没有意外增删接口。
// 新增一条时要同步加到这里：
//   - GET /api/get/record/log：下载单条记录的完整输出（入库的 memo 是截断过的）
var wantRoutes = []string{
	"GET /",
	"GET /api/get/conf/copy",
	"GET /api/get/conf/del",
	"GET /api/get/conf/get",
	"GET /api/get/conf/groupinfo",
	"GET /api/get/conf/list",
	"GET /api/get/conf/lock",
	"GET /api/get/conf/mylist",
	"GET /api/get/conf/server_groups",
	"GET /api/get/conf/tags",
	"GET /api/get/git/branch",
	"GET /api/get/git/commit",
	"GET /api/get/git/gitlog",
	"GET /api/get/git/gitpull",
	"GET /api/get/git/tag",
	"GET /api/get/jenkins/commit",
	"GET /api/get/other/noauto",
	"GET /api/get/p2p/check",
	"GET /api/get/p2p/send",
	"GET /api/get/p2p/task",
	"GET /api/get/record/attempts",
	"GET /api/get/record/list",
	"GET /api/get/record/log",
	"GET /api/get/task/changes",
	"GET /api/get/task/errlog",
	"GET /api/get/task/chart",
	"GET /api/get/task/del",
	"GET /api/get/task/get",
	"GET /api/get/task/last",
	"GET /api/get/task/list",
	"GET /api/get/task/rollback",
	"GET /api/get/test/api",
	"GET /api/get/user",
	"GET /api/get/user/del",
	"GET /api/get/user/project",
	"GET /api/get/walle/detection",
	"GET /api/get/walle/detectionssh",
	"GET /api/get/walle/flush",
	"GET /api/get/walle/md5",
	"GET /api/get/walle/release",
	"GET /api/post/p2p/agent",
	"GET /loginbydocke",
	"GET /v1/task",
	"GET /v1/task/:id",
	"GET /v1/token",
	"POST /api/get/conf/get",
	"POST /api/post/conf/save",
	"POST /api/post/p2p/agent",
	"POST /api/post/task/save",
	"POST /changePasswd",
	"POST /login",
	"POST /logout",
	"POST /register",
	"POST /v1/task",
	"PUT /v1/task/:id",
	"DELETE /v1/task/:id",
}

func newTestEcho() *echo.Echo {
	e := echo.New()
	// handler 真正执行时会访问未初始化的数据库并 panic，这里兜住，
	// 从而把"被中间件拦下"与"进了 handler"两种结果区分开
	e.Use(middleware.Recover())
	Register(e)
	return e
}

func registeredRoutes(e *echo.Echo) []string {
	var got []string
	for _, r := range e.Routes() {
		if r.Method == echo.RouteNotFound {
			continue // echo 为带中间件的分组自动登记的兜底路由
		}
		got = append(got, r.Method+" "+r.Path)
	}
	sort.Strings(got)
	return got
}

func TestRegisterRoutesUnchanged(t *testing.T) {
	got := registeredRoutes(newTestEcho())
	want := append([]string(nil), wantRoutes...)
	sort.Strings(want)

	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("路由表与预期不一致\n实际:\n%s\n预期:\n%s",
			strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestInternalRoutesRequireLogin 逐条请求所有路由（不带任何凭据），
// 除白名单外都必须返回未登录响应，而不是进入 handler。
func TestInternalRoutesRequireLogin(t *testing.T) {
	e := newTestEcho()

	for _, route := range registeredRoutes(e) {
		route := route
		parts := strings.SplitN(route, " ", 2)
		method, path := parts[0], parts[1]
		path = strings.ReplaceAll(path, ":id", "1")

		t.Run(route, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
			blocked := rec.Code == http.StatusOK &&
				strings.Contains(rec.Body.String(), `"code":2`)

			switch {
			case noAuthRoutes[route] && blocked:
				t.Errorf("免登录路由被拦截: %s", rec.Body.String())
			case !noAuthRoutes[route] && !blocked:
				// /v1/* 走的是 JWT 中间件，未登录时返回 errcode 102
				if strings.HasPrefix(path, "/v1/") &&
					strings.Contains(rec.Body.String(), `"errcode":"102"`) {
					return
				}
				t.Errorf("未登录即可访问: status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}
