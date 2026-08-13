package tasks

import (
	"os"
	"path/filepath"
	"time"

	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
)

// record 表是每次发布持续写入的，历史上没有任何清理机制，
// 只会一直涨 —— 表越大，上线进度页的查询越慢，备份也越难受。
// 这里按保留天数定期删掉过期的记录与对应的完整日志文件。

const (
	// 单批删除条数。分批是为了避免一条 DELETE 锁住大量行，
	// 影响正在发布的任务写日志。
	cleanBatchSize = 2000
	// 单轮最多删多少批，防止极端情况下一直占着连接
	cleanMaxBatches = 50
	cleanInterval   = 24 * time.Hour
	// 启动后先等一会再跑，避开启动时的迁移与初始化
	cleanStartDelay = 10 * time.Minute
)

// StartRecordCleaner 启动后台清理循环。
//
// 默认值刻意取 0（不清理）：老部署用的是自己的 app.conf，里面不会有这个键，
// 如果默认就删，升级完十分钟后历史日志会被静默清掉，这种事不能替用户决定。
// 要启用就在 app.conf 里显式写 recordKeepDays。
func StartRecordCleaner() {
	keepDays := config.DefaultInt("recordKeepDays", 0)
	if keepDays <= 0 {
		logger.Info("未配置 recordKeepDays，上线日志自动清理未启用")
		return
	}
	go func() {
		time.Sleep(cleanStartDelay)
		for {
			CleanRecords(keepDays)
			time.Sleep(cleanInterval)
		}
	}()
	logger.Info("上线日志自动清理已启用，保留天数:", keepDays)
}

// CleanRecords 删除 keepDays 之前的 record 行与日志文件
func CleanRecords(keepDays int) {
	if keepDays <= 0 {
		return
	}
	deadline := time.Now().AddDate(0, 0, -keepDays).Unix()

	total := int64(0)
	for i := 0; i < cleanMaxBatches; i++ {
		// created_at 是秒级时间戳。历史数据里存在 created_at=0 的脏记录
		// （旧实现插入时不写这个字段），它们同样属于该清理的范围。
		affected, err := db.Exec(
			"DELETE FROM `record` WHERE `created_at` < ? ORDER BY `id` ASC LIMIT ?",
			deadline, cleanBatchSize)
		if err != nil {
			logger.Error("清理上线记录失败:", err)
			break
		}
		total += affected
		if affected < cleanBatchSize {
			break
		}
	}
	if total > 0 {
		logger.Info("已清理过期上线记录:", total, "条")
	}
	cleanTaskLogFiles(deadline)
}

// cleanTaskLogFiles 删除过期的完整日志文件，空目录一并移除
func cleanTaskLogFiles(deadline int64) {
	entries, err := os.ReadDir(components.TaskLogRoot)
	if err != nil {
		// 目录不存在说明还没产生过日志，不是错误
		if !os.IsNotExist(err) {
			logger.Error("读取上线日志目录失败:", err)
		}
		return
	}
	removed := 0
	for _, dir := range entries {
		if !dir.IsDir() {
			continue
		}
		dirPath := filepath.Join(components.TaskLogRoot, dir.Name())
		files, err := os.ReadDir(dirPath)
		if err != nil {
			logger.Error("读取上线日志目录失败:", dirPath, err)
			continue
		}
		left := 0
		for _, f := range files {
			info, err := f.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Unix() >= deadline {
				left++
				continue
			}
			if err := os.Remove(filepath.Join(dirPath, f.Name())); err != nil {
				logger.Error("删除上线日志文件失败:", err)
				left++
				continue
			}
			removed++
		}
		if left == 0 {
			// 忽略错误：目录非空（并发写入）时删不掉是正常的
			os.Remove(dirPath)
		}
	}
	if removed > 0 {
		logger.Info("已清理过期上线日志文件:", removed, "个")
	}
}
