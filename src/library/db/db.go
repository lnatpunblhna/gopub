// Package db 封装 GORM 连接，并提供与 beego orm 原始查询语义等价的兼容层。
//
// 兼容层存在的原因：beego 的 Raw().Values()/ValuesList() 用 sql.NullString 扫描每一列
// （见 beego orm/orm_raw.go:664-694），因此结果里所有值都是 string，NULL 为 nil。
// 若直接用 GORM 扫描到 map，得到的是驱动原生类型（int64、[]uint8、time.Time），
// 序列化成 JSON 后字段类型会变化（字符串列甚至会变成 base64），前端将无法解析。
// 故这里精确复刻 beego 的行为。
package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Params 对应 beego 的 orm.Params
type Params = map[string]interface{}

// ParamsList 对应 beego 的 orm.ParamsList
type ParamsList = []interface{}

// ErrNoRows 对应 beego 的 orm.ErrNoRows
var ErrNoRows = gorm.ErrRecordNotFound

var gdb *gorm.DB

// DB 返回全局 GORM 句柄，替代 beego 的 orm.NewOrm()
func DB() *gorm.DB {
	return gdb
}

// 连接池默认值。与 README「连接池（默认 30 / 100）」一致。
// database/sql 自身的默认值是 MaxIdleConns=2、MaxOpenConns=无限，
// 前者会让连接用完即关（拿不到长连接复用的好处），后者在高并发时能打满
// MySQL 的 max_connections，两个都不适合本服务，因此这里显式给默认值。
const (
	defaultMaxIdleConn = 30
	defaultMaxOpenConn = 100
	// 连接最长存活时间。设得比 MySQL/中间层的 wait_timeout 短，
	// 让客户端主动淘汰连接，避免用到已被服务端单方面关闭的连接。
	defaultConnMaxLifetime = time.Hour
	// 空闲连接最长保留时间。比 lifetime 短，闲时能把池收缩回去，
	// 又足够长到让连续操作复用同一条连接。
	defaultConnMaxIdleTime = 10 * time.Minute
)

// PoolConfig 是连接池参数。零值字段一律回落到上面的默认值，
// 因此调用方只需覆盖关心的项。
type PoolConfig struct {
	MaxIdleConn     int
	MaxOpenConn     int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// withDefaults 把零值字段补成默认值
func (p PoolConfig) withDefaults() PoolConfig {
	if p.MaxIdleConn <= 0 {
		p.MaxIdleConn = defaultMaxIdleConn
	}
	if p.MaxOpenConn <= 0 {
		p.MaxOpenConn = defaultMaxOpenConn
	}
	if p.ConnMaxLifetime <= 0 {
		p.ConnMaxLifetime = defaultConnMaxLifetime
	}
	if p.ConnMaxIdleTime <= 0 {
		p.ConnMaxIdleTime = defaultConnMaxIdleTime
	}
	// 空闲上限大于连接上限没有意义：database/sql 会把 MaxIdleConns
	// 静默压到 MaxOpenConns，这里提前对齐，免得 Stats 看着对不上。
	if p.MaxIdleConn > p.MaxOpenConn {
		p.MaxIdleConn = p.MaxOpenConn
	}
	return p
}

// Init 建立数据库连接池。dsn 不含 parseTime 时会自动补上：
// GORM 依赖 go-sql-driver 的 parseTime=true 才能把 DATETIME 扫描进 time.Time，
// 而 beego orm 是自行转换的，原 DSN 里没有这个参数。
//
// *gorm.DB 底层是 database/sql 的连接池，全局复用一个即可：连接在池中长期保持，
// 每次查询借出、用完归还，不存在一次请求一次握手。
func Init(dsn string, pool PoolConfig, debug bool) error {
	level := gormlogger.Silent
	if debug {
		level = gormlogger.Info
	}

	var err error
	gdb, err = gorm.Open(mysqlDriver.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(level),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 表名不加复数，与原 beego 模型的 TableName() 保持一致
		},
	})
	if err != nil {
		return err
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	pool = pool.withDefaults()
	sqlDB.SetMaxIdleConns(pool.MaxIdleConn)
	sqlDB.SetMaxOpenConns(pool.MaxOpenConn)
	sqlDB.SetConnMaxLifetime(pool.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(pool.ConnMaxIdleTime)

	if err := sqlDB.Ping(); err != nil {
		return err
	}
	logger.Info("数据库连接池就绪: 最大连接", pool.MaxOpenConn, "最大空闲", pool.MaxIdleConn,
		"连接存活", pool.ConnMaxLifetime, "空闲存活", pool.ConnMaxIdleTime)
	return nil
}

// Stats 返回连接池实时状态，用于排查连接不够用/泄漏。
// 未初始化时返回零值。
func Stats() sql.DBStats {
	if gdb == nil {
		return sql.DBStats{}
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return sql.DBStats{}
	}
	return sqlDB.Stats()
}

// Close 关闭连接池，供进程退出时调用
func Close() error {
	if gdb == nil {
		return nil
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// BuildDSN 按原 beego 的连接串格式拼装 DSN，并补上 GORM 需要的 parseTime。
// timeout 只约束建立连接的握手阶段，不设 readTimeout/writeTimeout：
// 上线记录等查询耗时不可控，读超时会把正常的慢查询误杀成连接错误。
func BuildDSN(user, pass, host, port, name string) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true&loc=Asia%%2FShanghai&timeout=10s",
		user, pass, host, port, name,
	)
}

// MySQLConfig 返回最终生效的数据库连接参数。
// docker 运行模式下 MYSQL_* 环境变量优先于配置文件，与原 beego 版本行为一致。
func MySQLConfig() (user, pass, host, port, name string) {
	user = config.String("mysqluser")
	pass = config.String("mysqlpass")
	host = config.String("mysqlhost")
	port = config.String("mysqlport")
	name = config.String("mysqldb")

	if config.RunMode() != "docker" {
		return
	}
	for env, target := range map[string]*string{
		"MYSQL_USER": &user, "MYSQL_PASS": &pass, "MYSQL_HOST": &host,
		"MYSQL_PORT": &port, "MYSQL_DB": &name,
	} {
		if v := os.Getenv(env); v != "" {
			*target = v
		}
	}
	return
}

// PoolConfigFromConfig 读取 conf/app.conf 里的连接池配置。
// 键名以 db_ 为前缀，与 app.conf、README 保持一致；缺项走 PoolConfig 的默认值。
func PoolConfigFromConfig() PoolConfig {
	return PoolConfig{
		MaxIdleConn:     config.DefaultInt("db_max_idle_conn", defaultMaxIdleConn),
		MaxOpenConn:     config.DefaultInt("db_max_open_conn", defaultMaxOpenConn),
		ConnMaxLifetime: time.Duration(config.DefaultInt("db_conn_max_lifetime", int(defaultConnMaxLifetime/time.Second))) * time.Second,
		ConnMaxIdleTime: time.Duration(config.DefaultInt("db_conn_max_idle_time", int(defaultConnMaxIdleTime/time.Second))) * time.Second,
	}
}

// InitFromConfig 依据 conf/app.conf 中的配置建立连接。
// 幂等：docker 模式下建表流程与主流程都会调用，重复调用不会重建连接池。
func InitFromConfig() error {
	if gdb != nil {
		return nil
	}
	user, pass, host, port, name := MySQLConfig()
	return Init(BuildDSN(user, pass, host, port, name), PoolConfigFromConfig(), config.RunMode() == "dev")
}

// mysqlDatetimeLayout 是 MySQL DATETIME 列的原始文本格式。
// beego 的 DSN 不带 parseTime，驱动按原始字节返回时间列，因此结果里是
// "2006-01-02 15:04:05" 这种形式；本包为了让 GORM 能把时间列扫进
// model 的 time.Time 字段而启用了 parseTime，驱动会改为返回 time.Time，
// 这里再格式化回原始文本，保证接口返回给前端的时间格式不变。
const mysqlDatetimeLayout = "2006-01-02 15:04:05"

// normalize 把驱动返回的原生值统一成 string，NULL 保持为 nil，
// 与 beego 用 sql.NullString 扫描每一列的效果一致。
func normalize(v interface{}) interface{} {
	switch t := v.(type) {
	case nil:
		return nil
	case []byte:
		return string(t)
	case string:
		return t
	case time.Time:
		return t.Format(mysqlDatetimeLayout)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// scanRows 逐列扫描并把值统一成 string/nil，复刻 beego 的取值语义
func scanRows(rows *sql.Rows, fn func(cols []string, vals []interface{})) error {
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	for rows.Next() {
		refs := make([]interface{}, len(cols))
		holders := make([]interface{}, len(cols))
		for i := range refs {
			refs[i] = &holders[i]
		}
		if err := rows.Scan(refs...); err != nil {
			return err
		}
		vals := make([]interface{}, len(cols))
		for i, h := range holders {
			vals[i] = normalize(h)
		}
		fn(cols, vals)
	}
	return rows.Err()
}

// Values 等价于 beego 的 Raw(query, args...).Values(&container)
func Values(query string, args ...interface{}) ([]Params, error) {
	rows, err := gdb.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	var result []Params
	err = scanRows(rows, func(cols []string, vals []interface{}) {
		p := make(Params, len(cols))
		for i, c := range cols {
			p[c] = vals[i]
		}
		result = append(result, p)
	})
	return result, err
}

// ValuesList 等价于 beego 的 Raw(query, args...).ValuesList(&container)
func ValuesList(query string, args ...interface{}) ([]ParamsList, error) {
	rows, err := gdb.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	var result []ParamsList
	err = scanRows(rows, func(cols []string, vals []interface{}) {
		result = append(result, ParamsList(vals))
	})
	return result, err
}

// ValuesFlat 等价于 beego 的 Raw(query, args...).ValuesFlat(&container)，只取第一列
func ValuesFlat(query string, args ...interface{}) (ParamsList, error) {
	rows, err := gdb.Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	var result ParamsList
	err = scanRows(rows, func(cols []string, vals []interface{}) {
		if len(vals) > 0 {
			result = append(result, vals[0])
		}
	})
	return result, err
}

// QueryRow 等价于 beego 的 Raw(...).QueryRow(&dest)：无结果时返回 ErrNoRows。
// GORM 的 Scan 查不到数据不报错，这里补上以保持原语义。
func QueryRow(dest interface{}, query string, args ...interface{}) error {
	tx := gdb.Raw(query, args...).Scan(dest)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return ErrNoRows
	}
	return nil
}

// QueryRows 等价于 beego 的 Raw(...).QueryRows(&dest)，返回读取到的行数
func QueryRows(dest interface{}, query string, args ...interface{}) (int64, error) {
	tx := gdb.Raw(query, args...).Scan(dest)
	return tx.RowsAffected, tx.Error
}

// Exec 等价于 beego 的 Raw(...).Exec()，返回受影响行数
func Exec(query string, args ...interface{}) (int64, error) {
	tx := gdb.Exec(query, args...)
	return tx.RowsAffected, tx.Error
}

// MustValues 在查询出错时记录日志并返回空结果，便于替换原先忽略 error 的调用点
func MustValues(query string, args ...interface{}) []Params {
	rows, err := Values(query, args...)
	if err != nil {
		logger.Error("SQL查询失败:", query, err.Error())
		return nil
	}
	return rows
}
