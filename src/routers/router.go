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

// adminRoutePaths 记录本次注册中挂了 RequireAdmin 的路由（"METHOD /path" 形式），
// 供 router_test.go 断言 admin 清单没被漏改。由 Register 在每次注册开头重置。
var adminRoutePaths []string

// Register 把所有路由挂到 Echo 实例上。
//
// HTTP 方法按原 beego 控制器实际定义的方法确定：beego 依据控制器上定义的
// Get/Post 方法分派请求，未定义的方法返回 405，这里按同样的对应关系注册。
func Register(e *echo.Echo) {
	adminRoutePaths = nil

	// 跨域配置与原 beego cors 插件保持一致
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{"Origin", "UserToken", "Authorization", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type"},
		ExposeHeaders: []string{"Content-Length", "Access-Control-Allow-Origin",
			"Access-Control-Allow-Headers", "Content-Type"},
		MaxAge: 300, // 5 分钟
	}))

	// ---- 无需登录：只有登录入口与前端入口页 ----
	// 曾经这里还有一份「免登录只读白名单」，支撑 searchtaskList /
	// searchtaskRelease 两个匿名查询页。那两个页面与 taskList / taskRelease
	// 功能重合，免登录是它们存在的唯一理由，同时也是绕过下面 admin 门禁的口子，
	// 因此连页面带白名单一并去掉了。
	e.POST("/login", controllers.Login)
	e.GET("/loginbydocke", controllers.LoginByDocker)
	e.GET("/", controllers.Index)

	// 逐路由挂中间件而不用 e.Group("", ...)：空前缀分组会附带注册
	// RouteNotFound 的 "" 与 "/*" catch-all（见 echo/group.go 的 Group.Use），
	// 那会把所有未匹配路径原本的 404 变成这里的未登录响应。
	authGET := func(path string, h echo.HandlerFunc) {
		e.GET(path, h, controllers.RequireLogin)
	}
	authPOST := func(path string, h echo.HandlerFunc) {
		e.POST(path, h, controllers.RequireLogin)
	}
	// adminGET / adminPOST 对应前端菜单里标了 adminOnly 的那些页面。
	// 这里的判断依据是「接口的全部调用方都是 admin 页面」，逐个核对过
	// frontend/src/pages 下的引用；被普通用户页面用到的接口一律留在 authXXX。
	adminGET := func(path string, h echo.HandlerFunc) {
		e.GET(path, h, controllers.RequireAdmin)
		adminRoutePaths = append(adminRoutePaths, http.MethodGet+" "+path)
	}
	adminPOST := func(path string, h echo.HandlerFunc) {
		e.POST(path, h, controllers.RequireAdmin)
		adminRoutePaths = append(adminRoutePaths, http.MethodPost+" "+path)
	}

	// ---- 需要登录 ----
	authPOST("/logout", controllers.Logout)
	authPOST("/changePasswd", controllers.ChangePasswd)

	// 部署上线页（普通用户可用）：拉取项目展示信息、发布、看日志
	authGET("/api/get/conf/get", confcontrollers.Conf)
	authGET("/api/get/conf/mylist", confcontrollers.MyList)
	authGET("/api/get/conf/lock", confcontrollers.Lock)
	authGET("/api/get/conf/groupinfo", confcontrollers.GroupInfo)
	authGET("/api/get/walle/release", wallecontrollers.Release)
	authGET("/api/get/walle/md5", wallecontrollers.GetMd5)

	// 创建上线单页（普通用户可用）
	authGET("/api/get/git/branch", wallecontrollers.Branch)
	authGET("/api/get/git/commit", wallecontrollers.Commit)
	authGET("/api/get/git/tag", wallecontrollers.Tag)
	authGET("/api/get/jenkins/commit", wallecontrollers.Jenkins)
	authPOST("/api/post/task/save", taskcontrollers.Save)

	// 我的上线单：只返回当前登录用户自己的单子，用户 ID 取自登录态
	authGET("/api/get/task/mylist", taskcontrollers.MyList)
	authGET("/api/get/task/get", taskcontrollers.Task)
	authGET("/api/get/task/chart", taskcontrollers.TaskChart)
	authGET("/api/get/task/changes", taskcontrollers.Changes)
	authGET("/api/get/task/errlog", taskcontrollers.ErrLog)
	authGET("/api/get/task/last", taskcontrollers.LastTask)
	authGET("/api/get/task/rollback", taskcontrollers.RollBack)
	authGET("/api/get/task/del", taskcontrollers.Del)

	authGET("/api/get/record/list", recordcontrollers.List)
	authGET("/api/get/record/attempts", recordcontrollers.Attempts)
	// 完整日志里有命令、路径与目标机输出，必须登录且对该项目有权限才能下载
	authGET("/api/get/record/log", recordcontrollers.Log)

	// 改密码页要拿用户信息：handler 内部再判断「管理员或查自己」
	authGET("/api/get/user", usercontrollers.User)
	authGET("/api/get/test/api", controllers.TestApi)

	// ---- 需要管理员（Role==1）----
	// 用户管理页
	adminPOST("/register", controllers.Register)
	adminGET("/api/get/user/del", usercontrollers.Del)
	adminGET("/api/get/user/project", usercontrollers.UserProject)

	// 项目配置页
	adminGET("/api/get/conf/list", confcontrollers.List)
	adminPOST("/api/post/conf/save", confcontrollers.Save)
	adminPOST("/api/get/conf/get", confcontrollers.ConfSave)
	adminGET("/api/get/conf/del", confcontrollers.Del)
	adminGET("/api/get/conf/copy", confcontrollers.Copy)
	adminGET("/api/get/conf/tags", confcontrollers.Tags)
	adminGET("/api/get/conf/server_groups", confcontrollers.ServerGroups)
	adminGET("/api/get/walle/detection", wallecontrollers.Detection)
	adminGET("/api/get/walle/detectionssh", wallecontrollers.Detectionssh)

	// 全部上线单页
	adminGET("/api/get/task/list", taskcontrollers.List)

	// 运维工具页
	adminGET("/api/get/walle/flush", wallecontrollers.Flush)
	adminGET("/api/get/git/gitpull", wallecontrollers.Gitpull)
	adminGET("/api/get/git/gitlog", wallecontrollers.Gitlog)
	adminGET("/api/get/other/noauto", othercontrollers.NoAuto)
	adminGET("/api/get/p2p/task", p2pcontrollers.Task)
	adminGET("/api/get/p2p/check", p2pcontrollers.Check)
	adminGET("/api/get/p2p/send", p2pcontrollers.SendAgent)
	adminGET("/api/post/p2p/agent", p2pcontrollers.Agent)
	adminPOST("/api/post/p2p/agent", p2pcontrollers.AgentStart)

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
