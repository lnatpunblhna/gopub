package models

import (
	"fmt"
	"strings"

	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
)

// record 的 scope：区分同一张表里几类完全不同的记录。
//
// 改造前这张表混着四种东西：真实上线（task_id>0）、环境检测（-1）、缓存刷新（-2）、
// git pull（-3），以及列分支 / tag 这类只读查询（task.Id 为 0 时落到 -99）。
// 后面几类共用同一个 task_id，既永不清理，又让所有人的输出串在一起。
// 现在按 scope + user_id 区分，只读查询则完全不再落库。
const (
	RecordScopeRelease  = "release"  // 上线
	RecordScopeRollback = "rollback" // 回滚
	RecordScopeDetect   = "detect"   // 环境检测
	RecordScopeFlush    = "flush"    // 缓存刷新
	RecordScopeGitPull  = "gitpull"  // 目标机 git pull / git log
	RecordScopeAgent    = "agent"    // p2p agent 下发
)

// 检测 / 刷新 / gitpull 这类操作没有真实上线单，前端固定用这几个负数
// 占位 taskId 来取日志（frontend/src/pages/... 里的 <terminal :taskId="-1">）。
// 保留这套约定，但查询时不再只按 task_id 过滤，而是按 scope + 操作人隔离。
const (
	PseudoTaskDetect  = -1
	PseudoTaskFlush   = -2
	PseudoTaskGitPull = -3
	PseudoTaskAgent   = -4
	// PseudoTaskUnknown 兜底：既没有上线单、scope 又对不上任何一类的记录。
	// 正常不该出现，落在这里是为了出问题时还能查到，而不是混进别人的日志。
	PseudoTaskUnknown = -99
)

// ScopeByPseudoTaskId 把前端传来的占位 taskId 映射成 scope，未知则返回空串。
func ScopeByPseudoTaskId(taskId int64) string {
	switch taskId {
	case PseudoTaskDetect:
		return RecordScopeDetect
	case PseudoTaskFlush:
		return RecordScopeFlush
	case PseudoTaskGitPull:
		return RecordScopeGitPull
	case PseudoTaskAgent:
		return RecordScopeAgent
	}
	return ""
}

type Record struct {
	Id        int    `gorm:"column:id;primaryKey;autoIncrement"`
	UserId    uint   `gorm:"column:user_id;index:idx_record_scope_user,priority:2"`
	TaskId    int64  `gorm:"column:task_id;index:idx_record_task"`
	ProjectId int    `gorm:"column:project_id;not null;default:0"`
	Scope     string `gorm:"column:scope;type:varchar(16);not null;default:'';index:idx_record_scope_user,priority:1"`
	// Attempt 是同一个上线单的第几次发布尝试。
	// 以前重新发布会先 DELETE 掉这个任务的全部记录，上一次为什么失败就此消失；
	// 现在改为递增 attempt 保留历史，页面默认只看最近一次。
	Attempt   int    `gorm:"column:attempt;not null;default:1"`
	Status    int16  `gorm:"column:status"`
	Action    uint   `gorm:"column:action"`
	Command   string `gorm:"column:command;type:text"`
	Duration  int    `gorm:"column:duration"`
	Memo      string `gorm:"column:memo;type:mediumtext"`
	CreatedAt int    `gorm:"column:created_at;autoCreateTime:false"`
}

// NextAttempt 返回某个上线单的下一个发布批次号。
// 历史数据没有 attempt（默认 0/1），这里取 MAX+1，保证新一轮一定更大。
func NextAttempt(taskId int64) int {
	var maxAttempt int
	if err := db.DB().Model(&Record{}).
		Where("task_id = ?", taskId).
		Select("COALESCE(MAX(attempt),0)").
		Scan(&maxAttempt).Error; err != nil {
		logger.Error("读取发布批次失败 taskId=", taskId, " ", err)
		return 1
	}
	return maxAttempt + 1
}

// MaxAttempt 返回某个上线单当前最新的批次号，没有记录时返回 0
func MaxAttempt(taskId int64) int {
	var maxAttempt int
	if err := db.DB().Model(&Record{}).
		Where("task_id = ?", taskId).
		Select("COALESCE(MAX(attempt),0)").
		Scan(&maxAttempt).Error; err != nil {
		logger.Error("读取发布批次失败 taskId=", taskId, " ", err)
		return 0
	}
	return maxAttempt
}

func (t *Record) TableName() string {
	return "record"
}

// AddRecord insert a new Record into database and returns
// last inserted Id on success.
func AddRecord(m *Record) (id int64, err error) {
	if err = db.DB().Create(m).Error; err != nil {
		return 0, err
	}
	return int64(m.Id), nil
}

// GetRecordById retrieves Record by Id. Returns error if
// Id doesn't exist
func GetRecordById(id int) (v *Record, err error) {
	v = &Record{}
	if err = db.DB().First(v, id).Error; err != nil {
		return nil, err
	}
	return v, nil
}

// GetAllRecord retrieves all Record matches certain condition. Returns empty list if
// no records exist
func GetAllRecord(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	return getAll[Record](query, fields, sortby, order, offset, limit)
}

// UpdateRecord updates Record by Id and returns error if
// the record to be updated doesn't exist
func UpdateRecordById(m *Record) (err error) {
	// 先确认记录存在，保持原 beego 版本"记录不存在则返回错误"的语义
	if err = db.DB().First(&Record{}, m.Id).Error; err != nil {
		return err
	}
	tx := db.DB().Save(m)
	if tx.Error == nil {
		fmt.Println("Number of records updated in database:", tx.RowsAffected)
	}
	return tx.Error
}

// DeleteRecord deletes Record by Id and returns error if
// the record to be deleted doesn't exist
func DeleteRecord(id int) (err error) {
	if err = db.DB().First(&Record{}, id).Error; err != nil {
		return err
	}
	tx := db.DB().Delete(&Record{}, id)
	if tx.Error == nil {
		fmt.Println("Number of records deleted in database:", tx.RowsAffected)
	}
	return tx.Error
}

// MigrateRecordSchema 补齐 record 表新增的列与索引。
//
// 与 MigrateAuthKeyColumn 同理：Syncdb 只在 -syncdb / -docker 下执行，
// 老部署直接换二进制不会有 scope / project_id 列，缺列会让所有日志写入失败。
// 索引尤其关键：没有 idx_record_task，进度页每 2 秒一次的查询就是全表扫描。
//
// memo 的类型只做检查不自动改：TEXT 改 MEDIUMTEXT 是一次 ALTER TABLE 重建，
// record 表大时可能锁很久，不能放在启动路径上，交由运维在窗口期手工执行。
func MigrateRecordSchema() {
	m := db.DB().Migrator()
	if !m.HasTable(&Record{}) {
		return
	}
	for _, col := range []string{"project_id", "scope", "attempt"} {
		if m.HasColumn(&Record{}, col) {
			continue
		}
		if err := m.AddColumn(&Record{}, col); err != nil {
			logger.Error("record."+col+" 列创建失败，上线日志将无法写入，请手动执行 ./control syncdb:", err)
			continue
		}
		logger.Info("已补齐 record." + col + " 列")
	}
	for _, idx := range []string{"idx_record_task", "idx_record_scope_user"} {
		if m.HasIndex(&Record{}, idx) {
			continue
		}
		if err := m.CreateIndex(&Record{}, idx); err != nil {
			logger.Error("record 索引 "+idx+" 创建失败，上线进度页会退化成全表扫描:", err)
			continue
		}
		logger.Info("已创建 record 索引 " + idx)
	}
	// memo 列宽度只提示不自动改。TEXT 上限 65535 字节，单步输出（尤其多机汇总）
	// 很容易撑爆，撑爆后整条 UPDATE 失败，页面上表现为「这一步没有任何输出」。
	// 入库前已有截断兜底，扩容与否交给运维决定。
	if MemoColumnIsSmall() {
		logger.Info("record.memo 仍是 TEXT(65535 字节)，单步输出超限会被截断；" +
			"可在维护窗口执行 ALTER TABLE `record` MODIFY `memo` MEDIUMTEXT 以保留更完整的日志")
	}
}

// MemoColumnIsSmall 返回 memo 列是否仍是 TEXT（65535 字节上限）。
func MemoColumnIsSmall() bool {
	types, err := db.DB().Migrator().ColumnTypes(&Record{})
	if err != nil {
		return false
	}
	for _, t := range types {
		if t.Name() != "memo" {
			continue
		}
		name := strings.ToUpper(t.DatabaseTypeName())
		return name != "MEDIUMTEXT" && name != "LONGTEXT"
	}
	return false
}
