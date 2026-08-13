package main

import (
	"context"
	"html/template"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/library/p2p/init_sever"
	"github.com/lnatpunblhna/gopub/src/models"
	"github.com/lnatpunblhna/gopub/src/routers"
	"github.com/lnatpunblhna/gopub/src/tasks"
)

func initArgs() {
	for _, v := range os.Args {
		if v == "-syncdb" {
			models.Syncdb()
			os.Exit(0)
		}
		if v == "-docker" {
			config.SetRunMode("docker")
			models.Syncdb()
		}
	}
}

// initLogger 设置日志:按天分割,保留30天,自动轮转
func initLogger() {
	fn := "logs/run.log"
	os.MkdirAll("logs", 0755)
	if err := logger.SetFile(fn, 30); err != nil {
		println("日志文件初始化失败: " + err.Error())
	}
	if config.RunMode() == "prod" {
		logger.SetLevel(logger.LevelInformational)
	}
}

func init() {
	initLogger()
	logger.Info("开始启动")
	//初始化数据库
	initArgs()
	//连接MySQL
	if os.Getenv("JenkinsUserName") != "" {
		config.Set("JenkinsUserName", os.Getenv("JenkinsUserName"))
	}
	if os.Getenv("JenkinsPwd") != "" {
		config.Set("JenkinsPwd", os.Getenv("JenkinsPwd"))
	}
	if err := db.InitFromConfig(); err != nil {
		logger.Error("数据库连接错误:", err)
		os.Exit(2)
	}
	// 老部署直接换二进制时 Syncdb 不会执行，这里补齐登录凭据的过期时间列
	models.MigrateAuthKeyColumn()
	// 同理补齐 record 表的 scope / project_id 列与查询索引，
	// 缺索引会让上线进度页每次轮询都全表扫描
	models.MigrateRecordSchema()
}

// tplRenderer 渲染 views 目录下的模板，替代 beego 的模板自动渲染
type tplRenderer struct {
	templates *template.Template
}

func (t *tplRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	if t.templates == nil {
		return echo.NewHTTPError(500, "模板未加载: "+name)
	}
	return t.templates.ExecuteTemplate(w, name, data)
}

func newRenderer() *tplRenderer {
	tpl, err := template.ParseGlob("views/*.tpl")
	if err != nil {
		logger.Error("模板加载失败:", err)
		return &tplRenderer{}
	}
	return &tplRenderer{templates: tpl}
}

func main() {
	//获取全局panic
	defer func() {
		if err := recover(); err != nil {
			logger.Error("Panic error:", err)
		}
	}()

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	if enableGzip, _ := config.Bool("EnableGzip"); enableGzip {
		e.Use(middleware.Gzip())
	}
	if accessLogs, _ := config.Bool("AccessLogs"); accessLogs {
		e.Use(middleware.Logger())
	}
	// 使 /path/ 与 /path 等价，对应 beego 命名空间路由的行为
	e.Pre(middleware.RemoveTrailingSlash())

	e.Renderer = newRenderer()
	e.Static("/static", "static")
	e.File("/favicon.ico", "favicon.ico")

	//API自动化文档
	if config.RunMode() == "dev" {
		e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
			Root:   "swagger",
			Browse: true,
		}))
	}

	routers.Register(e)

	logger.Info(config.RunMode())
	if config.RunMode() != "docker" {
		init_sever.Start()
	}
	// 上线日志按保留天数定期清理，否则 record 表只会一直涨
	tasks.StartRecordCleaner()

	addr := config.DefaultString("HttpAddr", "0.0.0.0") + ":" + config.DefaultString("httpport", "8080")
	//热启动
	graceful, _ := config.Bool("Graceful")
	go handleSignals(e, graceful)

	logger.Info("监听地址:", addr)
	if err := e.Start(addr); err != nil {
		logger.Info("Shutdown, bye...", err)
	}
}

// handleSignals 处理退出信号；Graceful 开启时等待在途请求处理完毕
func handleSignals(e *echo.Echo, graceful bool) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-sigs

	if !graceful {
		logger.Info("Shutdown quickly, bye...", sig)
		logger.Close()
		os.Exit(0)
	}

	logger.Info("Shutdown gracefully, bye...", sig)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		logger.Error("优雅关闭失败:", err)
	}
	logger.Close()
	os.Exit(0)
}
