package models

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
)

func Syncdb() {
	logger.Info("数据库初始化开始")
	err := createdb()
	if err != nil {
		logger.Error("数据库创建错误:", err)
		return
	}

	Connect()
	// 建表，替代 beego 的 orm.RunSyncdb
	if err = db.DB().AutoMigrate(AllModels()...); err != nil {
		logger.Error("数据表创建错误:", err)
	}
	logger.Info("数据表创建完成")
	insertUser()
	logger.Info("数据添加完成")
}

// 数据库连接
func Connect() {
	if err := db.InitFromConfig(); err != nil {
		logger.Error("数据库连接错误:", err)
		os.Exit(2)
	}
}

// 创建数据库。
// 这里刻意不复用全局连接池：建库时目标库还不存在，DSN 不能带库名，
// 用完即弃，因此把池压到 1 条连接。
func createdb() error {
	dbUser, dbPass, dbHost, dbPort, dbName := db.MySQLConfig()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8&timeout=10s", dbUser, dbPass, dbHost, dbPort)
	conn, err := sql.Open("mysql", dsn)
	if err != nil {
		logger.Error("数据库连接错误:", err)
		os.Exit(2)
		return err
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	// sql.Open 只解析 DSN 不建连，真正的连通性要靠 Ping 才能暴露出来
	if err = conn.Ping(); err != nil {
		logger.Error("数据库连接错误:", err)
		os.Exit(2)
		return err
	}

	sqlstring := fmt.Sprintf(" CREATE DATABASE if not exists `%s` CHARSET utf8 COLLATE utf8_general_ci", dbName)
	r, err := conn.Exec(sqlstring)
	if err != nil {
		logger.Info(err)
		logger.Info(r)
		return err
	}
	logger.Info("数据库" + dbName + "创建成功")
	return nil
}

func insertUser() {
	fmt.Println("insert user ...")
	u := new(User)
	u.Id = 1
	u.Username = "admin"
	u.IsEmailVerified = 1
	// 不预置 auth_key：写死的初始值等于把一个公开的 admin 凭据装进每套部署里。
	// 留空后由首次登录签发（models.IssueAuthKey）。
	u.AuthKey = ""
	u.PasswordHash = "$2y$13$8q0MfKpnghuqCL.3FAAjiOkA8kBFNCW.ECUlqWp1zTpMHs9e5xn6u"
	u.EmailConfirmationToken = "UpToOIawm1L8GjN6pLO4r-1oj20nLT5f_1443280741"
	u.Email = "lnatpunblhna@gmail.com"
	u.Avatar = "default.jpg"
	u.Role = 1
	u.Status = 10
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	u.Realname = "管理员"
	db.DB().Create(u)
	fmt.Println("insert user end")
}
