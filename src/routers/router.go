// Package routers 注册全部 HTTP 路由，替代原 beego 的 beego.Router / NewNamespace。
package routers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/linclin/gopub/src/controllers"
	apicontrollers "github.com/linclin/gopub/src/controllers/api"
	confcontrollers "github.com/linclin/gopub/src/controllers/conf"
	othercontrollers "github.com/linclin/gopub/src/controllers/other"
	p2pcontrollers "github.com/linclin/gopub/src/controllers/p2p"
	recordcontrollers "github.com/linclin/gopub/src/controllers/record"
	taskcontrollers "github.com/linclin/gopub/src/controllers/task"
	usercontrollers "github.com/linclin/gopub/src/controllers/user"
	wallecontrollers "github.com/linclin/gopub/src/controllers/walle"
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

	e.POST("/login", controllers.Login)
	e.POST("/logout", controllers.Logout)
	e.GET("/loginbydocke", controllers.LoginByDocker)
	e.POST("/changePasswd", controllers.ChangePasswd)
	e.POST("/register", controllers.Register)

	e.GET("/api/get/walle/detection", wallecontrollers.Detection)
	e.GET("/api/get/walle/detectionssh", wallecontrollers.Detectionssh)
	e.GET("/api/get/walle/release", wallecontrollers.Release)
	e.GET("/api/get/walle/md5", wallecontrollers.GetMd5)
	e.GET("/api/get/walle/flush", wallecontrollers.Flush)

	e.GET("/api/get/conf/list", confcontrollers.List)
	e.GET("/api/get/conf/mylist", confcontrollers.MyList)
	e.GET("/api/get/conf/get", confcontrollers.Conf)
	e.POST("/api/get/conf/get", confcontrollers.ConfSave)
	e.POST("/api/post/conf/save", confcontrollers.Save)
	e.GET("/api/get/conf/del", confcontrollers.Del)
	e.GET("/api/get/conf/copy", confcontrollers.Copy)
	e.GET("/api/get/conf/tags", confcontrollers.Tags)
	e.GET("/api/get/conf/lock", confcontrollers.Lock)
	e.GET("/api/get/conf/server_groups", confcontrollers.ServerGroups)
	e.GET("/api/get/conf/groupinfo", confcontrollers.GroupInfo)

	e.GET("/api/get/git/branch", wallecontrollers.Branch)
	e.GET("/api/get/git/commit", wallecontrollers.Commit)
	e.GET("/api/get/git/gitpull", wallecontrollers.Gitpull)
	e.GET("/api/get/git/gitlog", wallecontrollers.Gitlog)
	e.GET("/api/get/git/tag", wallecontrollers.Tag)

	e.GET("/api/get/jenkins/commit", wallecontrollers.Jenkins)

	e.GET("/api/get/task/list", taskcontrollers.List)
	e.GET("/api/get/task/chart", taskcontrollers.TaskChart)
	e.POST("/api/post/task/save", taskcontrollers.Save)
	e.GET("/api/get/task/get", taskcontrollers.Task)
	e.GET("/api/get/task/changes", taskcontrollers.Changes)
	e.GET("/api/get/task/last", taskcontrollers.LastTask)
	e.GET("/api/get/task/rollback", taskcontrollers.RollBack)
	e.GET("/api/get/task/del", taskcontrollers.Del)

	e.GET("/api/get/p2p/task", p2pcontrollers.Task)
	e.GET("/api/get/p2p/check", p2pcontrollers.Check)
	e.GET("/api/post/p2p/agent", p2pcontrollers.Agent)
	e.POST("/api/post/p2p/agent", p2pcontrollers.AgentStart)
	e.GET("/api/get/p2p/send", p2pcontrollers.SendAgent)

	e.GET("/api/get/record/list", recordcontrollers.List)

	e.GET("/api/get/other/noauto", othercontrollers.NoAuto)
	e.GET("/api/get/test/api", controllers.TestApi)
	e.GET("/api/get/user/project", usercontrollers.UserProject)
	e.GET("/api/get/user/del", usercontrollers.Del)
	e.GET("/api/get/user", usercontrollers.User)
	e.GET("/", controllers.Index)

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
