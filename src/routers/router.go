// Package routers 注册全部 HTTP 路由，替代原 beego 的 beego.Router / NewNamespace。
package routers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lnatpunblhna/gopub/src/controllers"
	apicontrollers "github.com/lnatpunblhna/gopub/src/controllers/api"
	confcontrollers "github.com/lnatpunblhna/gopub/src/controllers/conf"
	othercontrollers "github.com/lnatpunblhna/gopub/src/controllers/other"
	p2pcontrollers "github.com/lnatpunblhna/gopub/src/controllers/p2p"
	recordcontrollers "github.com/lnatpunblhna/gopub/src/controllers/record"
	taskcontrollers "github.com/lnatpunblhna/gopub/src/controllers/task"
	usercontrollers "github.com/lnatpunblhna/gopub/src/controllers/user"
	wallecontrollers "github.com/lnatpunblhna/gopub/src/controllers/walle"
)

// Register 把所有路由挂到 Echo 实例上。
//
// HTTP 方法按原 beego 控制器实际定义的方法确定：beego 依据控制器上定义的
// Get/Post 方法分派请求，未定义的方法返回 405，这里按同样的对应关系注册。
func Register(e *echo.Echo) {
	// 跨域配置与原 beego cors 插件保持一致
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{"Origin", "UserToken", "Authorization", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type"},
		ExposeHeaders: []string{"Content-Length", "Access-Control-Allow-Origin",
			"Access-Control-Allow-Headers", "Content-Type"},
		MaxAge: 300, // 5 分钟
	}))

	// ---- 无需登录：登录入口与前端入口页 ----
	e.POST("/login", controllers.Login)
	e.GET("/loginbydocke", controllers.LoginByDocker)
	e.GET("/", controllers.Index)

	// ---- 免登录只读白名单 ----
	// 前端 searchtaskList / searchtaskRelease 两个页面允许未登录访问
	// （frontend/src/router/index.js 里对这两个路由名放行），它们只用到下面
	// 四个查询接口。conf/get 对未登录请求会脱敏，见 confcontrollers.Conf。
	e.GET("/api/get/task/list", taskcontrollers.List)
	e.GET("/api/get/task/get", taskcontrollers.Task)
	e.GET("/api/get/conf/get", confcontrollers.Conf)
	e.GET("/api/get/record/list", recordcontrollers.List)
	// 只返回批次号 / 时间 / 成败，不含命令与主机信息，与 record/list 一样放行
	e.GET("/api/get/record/attempts", recordcontrollers.Attempts)

	// ---- 需要登录 ----
	// 逐路由挂中间件而不用 e.Group("", ...)：空前缀分组会附带注册
	// RouteNotFound 的 "" 与 "/*" catch-all（见 echo/group.go 的 Group.Use），
	// 那会把所有未匹配路径原本的 404 变成这里的未登录响应。
	authGET := func(path string, h echo.HandlerFunc) {
		e.GET(path, h, controllers.RequireLogin)
	}
	authPOST := func(path string, h echo.HandlerFunc) {
		e.POST(path, h, controllers.RequireLogin)
	}

	authPOST("/logout", controllers.Logout)
	authPOST("/changePasswd", controllers.ChangePasswd)

	authGET("/api/get/walle/detection", wallecontrollers.Detection)
	authGET("/api/get/walle/detectionssh", wallecontrollers.Detectionssh)
	authGET("/api/get/walle/release", wallecontrollers.Release)
	authGET("/api/get/walle/md5", wallecontrollers.GetMd5)
	authGET("/api/get/walle/flush", wallecontrollers.Flush)

	authGET("/api/get/conf/list", confcontrollers.List)
	authGET("/api/get/conf/mylist", confcontrollers.MyList)
	authPOST("/api/get/conf/get", confcontrollers.ConfSave)
	authPOST("/api/post/conf/save", confcontrollers.Save)
	authGET("/api/get/conf/del", confcontrollers.Del)
	authGET("/api/get/conf/copy", confcontrollers.Copy)
	authGET("/api/get/conf/tags", confcontrollers.Tags)
	authGET("/api/get/conf/lock", confcontrollers.Lock)
	authGET("/api/get/conf/server_groups", confcontrollers.ServerGroups)
	authGET("/api/get/conf/groupinfo", confcontrollers.GroupInfo)

	authGET("/api/get/git/branch", wallecontrollers.Branch)
	authGET("/api/get/git/commit", wallecontrollers.Commit)
	authGET("/api/get/git/gitpull", wallecontrollers.Gitpull)
	authGET("/api/get/git/gitlog", wallecontrollers.Gitlog)
	authGET("/api/get/git/tag", wallecontrollers.Tag)

	authGET("/api/get/jenkins/commit", wallecontrollers.Jenkins)

	authGET("/api/get/task/chart", taskcontrollers.TaskChart)
	authPOST("/api/post/task/save", taskcontrollers.Save)
	authGET("/api/get/task/changes", taskcontrollers.Changes)
	authGET("/api/get/task/errlog", taskcontrollers.ErrLog)
	authGET("/api/get/task/last", taskcontrollers.LastTask)
	authGET("/api/get/task/rollback", taskcontrollers.RollBack)
	authGET("/api/get/task/del", taskcontrollers.Del)

	authGET("/api/get/p2p/task", p2pcontrollers.Task)
	authGET("/api/get/p2p/check", p2pcontrollers.Check)
	authGET("/api/post/p2p/agent", p2pcontrollers.Agent)
	authPOST("/api/post/p2p/agent", p2pcontrollers.AgentStart)
	authGET("/api/get/p2p/send", p2pcontrollers.SendAgent)

	// 完整日志里有命令、路径与目标机输出，必须登录且对该项目有权限才能下载
	authGET("/api/get/record/log", recordcontrollers.Log)

	authGET("/api/get/other/noauto", othercontrollers.NoAuto)
	authGET("/api/get/test/api", controllers.TestApi)
	authGET("/api/get/user/project", usercontrollers.UserProject)

	// ---- 需要管理员（Role==1）----
	// 建号 / 改号（/register 两种语义都走它）、删号、用户列表
	e.POST("/register", controllers.Register, controllers.RequireAdmin)
	e.GET("/api/get/user/del", usercontrollers.Del, controllers.RequireAdmin)
	e.GET("/api/get/user", usercontrollers.User, controllers.RequireAdmin)

	registerAPI(e)
}

// registerAPI 注册对外开放的 /v1 接口，对应原 beego 的 NewNamespace("/v1", ...)
func registerAPI(e *echo.Echo) {
	v1 := e.Group("/v1")

	// 签发 token 的接口本身不能要求携带 token
	v1.GET("/token", apicontrollers.IssueToken)

	// 其余接口需通过 JWT 校验，对应原 BaseApiController.Prepare
	task := v1.Group("/task", apicontrollers.ApiAuth)
	task.POST("", apicontrollers.TaskPost)
	// 原 URLMapping 把 "GetAll" 映射到了 GetAllAndProName，
	// 因此 GET /v1/task 对应的是按项目名与时间区间的查询
	task.GET("", apicontrollers.TaskGetAllAndProName)
	task.GET("/:id", apicontrollers.TaskGetOne)
	task.PUT("/:id", apicontrollers.TaskPut)
	task.DELETE("/:id", apicontrollers.TaskDelete)
}
