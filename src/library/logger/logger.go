// Package logger 提供与 beego 日志（beego.Info/Error/Debug 等）语义等价的输出能力。
//
// 与原 beego 配置保持一致：
//   - 同时输出到控制台与 logs/run.log（beego 默认带 console adapter，
//     main.go 再通过 SetLogger("file", ...) 追加文件 adapter）
//   - 文件按天切割、自动轮转、保留 30 天
//   - prod 模式下等级设为 Informational，屏蔽 Debug 输出
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// 日志级别，取值与 beego 的 LevelXxx 对齐（数值越小越严重）
const (
	LevelEmergency = iota
	LevelAlert
	LevelCritical
	LevelError
	LevelWarning
	LevelNotice
	LevelInformational
	LevelDebug
)

var levelTag = map[int]string{
	LevelEmergency:     "M",
	LevelAlert:         "A",
	LevelCritical:      "C",
	LevelError:         "E",
	LevelWarning:       "W",
	LevelNotice:        "N",
	LevelInformational: "I",
	LevelDebug:         "D",
}

var (
	mu       sync.Mutex
	level    = LevelDebug
	file     *os.File
	filePath string
	fileDay  string
	maxDays  = 30
)

// SetLevel 设置输出等级，等价于 beego.SetLevel
func SetLevel(l int) {
	mu.Lock()
	level = l
	mu.Unlock()
}

// SetFile 启用文件输出，等价于 beego.SetLogger("file", ...)。
// path 例如 logs/run.log；keepDays 为保留天数。
func SetFile(path string, keepDays int) error {
	mu.Lock()
	defer mu.Unlock()

	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	if file != nil {
		file.Close()
	}
	file, filePath, maxDays = f, path, keepDays
	fileDay = time.Now().Format("2006-01-02")
	return nil
}

// rotate 在跨天时把当前文件改名归档并新建，同时清理过期归档。调用方需持有 mu。
func rotate(now time.Time) {
	if file == nil {
		return
	}
	day := now.Format("2006-01-02")
	if day == fileDay {
		return
	}

	file.Close()
	ext := filepath.Ext(filePath)
	archived := strings.TrimSuffix(filePath, ext) + "." + fileDay + ext
	// 归档名已存在时直接续写原文件，避免丢日志
	if _, err := os.Stat(archived); err == nil {
		archived = strings.TrimSuffix(filePath, ext) + "." + fileDay + "." + now.Format("150405") + ext
	}
	os.Rename(filePath, archived)

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		file = nil
		return
	}
	file, fileDay = f, day
	cleanOldFiles()
}

// cleanOldFiles 只保留最近 maxDays 个归档文件。调用方需持有 mu。
func cleanOldFiles() {
	if maxDays <= 0 {
		return
	}
	ext := filepath.Ext(filePath)
	prefix := filepath.Base(strings.TrimSuffix(filePath, ext)) + "."
	dir := filepath.Dir(filePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var archives []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ext) {
			continue
		}
		archives = append(archives, name)
	}
	if len(archives) <= maxDays {
		return
	}
	sort.Strings(archives) // 文件名含日期，字典序即时间序
	for _, name := range archives[:len(archives)-maxDays] {
		os.Remove(filepath.Join(dir, name))
	}
}

// write 按 beego 的行格式输出：2006/01/02 15:04:05.000 [I] message
func write(l int, v ...interface{}) {
	mu.Lock()
	defer mu.Unlock()

	if l > level {
		return
	}
	now := time.Now()
	msg := fmt.Sprintf("%s [%s] %s\n", now.Format("2006/01/02 15:04:05.000"), levelTag[l], joinArgs(v))

	os.Stdout.WriteString(msg)
	rotate(now)
	if file != nil {
		file.WriteString(msg)
	}
}

// joinArgs 复刻 beego 多参数以空格拼接的行为
func joinArgs(v []interface{}) string {
	parts := make([]string, 0, len(v))
	for _, item := range v {
		parts = append(parts, fmt.Sprintf("%v", item))
	}
	return strings.Join(parts, " ")
}

// 以下函数与 beego.Xxx 一一对应

func Emergency(v ...interface{}) { write(LevelEmergency, v...) }
func Alert(v ...interface{})     { write(LevelAlert, v...) }
func Critical(v ...interface{})  { write(LevelCritical, v...) }
func Error(v ...interface{})     { write(LevelError, v...) }
func Warning(v ...interface{})   { write(LevelWarning, v...) }
func Warn(v ...interface{})      { write(LevelWarning, v...) }
func Notice(v ...interface{})    { write(LevelNotice, v...) }
func Info(v ...interface{})      { write(LevelInformational, v...) }
func Debug(v ...interface{})     { write(LevelDebug, v...) }

// Close 关闭文件句柄，供进程退出时调用
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		file.Close()
		file = nil
	}
}
