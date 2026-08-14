// Package config 提供与 beego.AppConfig 语义等价的配置读取能力。
//
// 保持 conf/app.conf 的 ini 格式与原有语义完全一致，替换 beego 后运维配置无需改动：
//  1. key 一律小写化后存取（beego config/ini.go:150 `key = strings.ToLower(key)`）
//  2. 值若以 " 开头则去除两端引号（beego config/ini.go:194）
//  3. String(key) 先查 `runmode::key`，取到空串再回落全局段（beego config.go:439）
//  4. Int/Bool(key) 先查 `runmode::key`，解析失败再回落全局段（beego config.go:453/467）
package config

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// 全局段名，对应 beego config 的 defaultSection
const defaultSection = "default"

var (
	mu      sync.RWMutex
	data    = map[string]map[string]string{}
	runMode = "dev"
)

func init() {
	if err := Load(""); err != nil {
		// 与 beego 一致：配置缺失不 panic，交由后续取值返回零值
		println("加载配置文件失败: " + err.Error())
	}
	if m := getRaw("runmode"); m != "" {
		SetRunMode(m)
	}
}

// Load 解析配置文件。path 为空时按 beego 的习惯查找 conf/app.conf：
// 先相对当前工作目录，再相对可执行文件所在目录。
func Load(path string) error {
	if path == "" {
		candidates := []string{filepath.Join("conf", "app.conf")}
		if exe, err := os.Executable(); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(exe), "conf", "app.conf"))
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				path = c
				break
			}
		}
		if path == "" {
			return errors.New("找不到配置文件 conf/app.conf，请执行 ./control start（会自动从 conf/app.conf.example 生成），或手动复制模板：cp conf/app.conf.example conf/app.conf")
		}
	}

	parsed, err := parseFile(path)
	if err != nil {
		return err
	}
	mu.Lock()
	data = parsed
	mu.Unlock()
	return nil
}

func parseFile(path string) (map[string]map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := map[string]map[string]string{defaultSection: {}}
	section := defaultSection

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// 整行注释：# 或 ;（与 beego 一致，不处理行内注释）
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		// 段声明 [name]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			if section == "" {
				section = defaultSection
			}
			if _, ok := out[section]; !ok {
				out[section] = map[string]string{}
			}
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:idx]))
		if key == "" {
			continue
		}
		val := strings.TrimSpace(line[idx+1:])
		if strings.HasPrefix(val, `"`) {
			val = strings.Trim(val, `"`)
		}
		out[section][key] = expandValueEnv(val)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// expandValueEnv 支持 beego 的 ${ENV||default} 取值语法
func expandValueEnv(val string) string {
	if !strings.HasPrefix(val, "${") || !strings.HasSuffix(val, "}") {
		return val
	}
	inner := val[2 : len(val)-1]
	key, defaultVal := inner, ""
	if i := strings.Index(inner, "||"); i >= 0 {
		key = strings.TrimSpace(inner[:i])
		defaultVal = strings.TrimSpace(inner[i+2:])
	}
	if key == "" {
		return val
	}
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// getRaw 按 "section::key" 或 "key"（落到全局段）取原始值，取不到返回空串
func getRaw(key string) string {
	if key == "" {
		return ""
	}
	mu.RLock()
	defer mu.RUnlock()

	section, k := defaultSection, strings.ToLower(key)
	if parts := strings.SplitN(k, "::", 2); len(parts) == 2 {
		section, k = parts[0], parts[1]
	}
	if sec, ok := data[section]; ok {
		if v, ok := sec[k]; ok {
			return v
		}
	}
	return ""
}

// RunMode 返回当前运行模式，替代 beego.BConfig.RunMode 的读取
func RunMode() string {
	mu.RLock()
	defer mu.RUnlock()
	return runMode
}

// SetRunMode 设置运行模式，替代对 beego.BConfig.RunMode 的赋值
func SetRunMode(mode string) {
	mu.Lock()
	runMode = mode
	mu.Unlock()
}

// String 等价于 beego.AppConfig.String：优先当前 runmode 段，空则回落全局段
func String(key string) string {
	if v := getRaw(RunMode() + "::" + key); v != "" {
		return v
	}
	return getRaw(key)
}

// DefaultString 取不到时返回 defaultVal
func DefaultString(key, defaultVal string) string {
	if v := String(key); v != "" {
		return v
	}
	return defaultVal
}

// Int 等价于 beego.AppConfig.Int：优先当前 runmode 段，解析失败则回落全局段
func Int(key string) (int, error) {
	if v, err := strconv.Atoi(getRaw(RunMode() + "::" + key)); err == nil {
		return v, nil
	}
	return strconv.Atoi(getRaw(key))
}

// DefaultInt 取不到或解析失败时返回 defaultVal
func DefaultInt(key string, defaultVal int) int {
	if v, err := Int(key); err == nil {
		return v
	}
	return defaultVal
}

// Int64 语义同 Int
func Int64(key string) (int64, error) {
	if v, err := strconv.ParseInt(getRaw(RunMode()+"::"+key), 10, 64); err == nil {
		return v, nil
	}
	return strconv.ParseInt(getRaw(key), 10, 64)
}

// Bool 等价于 beego.AppConfig.Bool：优先当前 runmode 段，解析失败则回落全局段
func Bool(key string) (bool, error) {
	if v, err := parseBool(getRaw(RunMode() + "::" + key)); err == nil {
		return v, nil
	}
	return parseBool(getRaw(key))
}

// DefaultBool 取不到或解析失败时返回 defaultVal
func DefaultBool(key string, defaultVal bool) bool {
	if v, err := Bool(key); err == nil {
		return v
	}
	return defaultVal
}

// Float 等价于 beego.AppConfig.Float
func Float(key string) (float64, error) {
	if v, err := strconv.ParseFloat(getRaw(RunMode()+"::"+key), 64); err == nil {
		return v, nil
	}
	return strconv.ParseFloat(getRaw(key), 64)
}

// Set 等价于 beego.AppConfig.Set：写入当前 runmode 段
func Set(key, val string) error {
	if key == "" {
		return errors.New("配置项名称不能为空")
	}
	mu.Lock()
	defer mu.Unlock()

	// 已持有写锁，直接读 runMode，不能再调用 RunMode()（RWMutex 不可重入）
	section, k := runMode, strings.ToLower(key)
	if parts := strings.SplitN(k, "::", 2); len(parts) == 2 {
		section, k = parts[0], parts[1]
	}
	section = strings.ToLower(section)
	if _, ok := data[section]; !ok {
		data[section] = map[string]string{}
	}
	data[section][k] = val
	return nil
}

// parseBool 复刻 beego config.ParseBool 支持的字面量集合
func parseBool(val string) (bool, error) {
	switch strings.TrimSpace(val) {
	case "1", "t", "T", "true", "TRUE", "True", "YES", "yes", "Yes", "Y", "y", "ON", "on", "On":
		return true, nil
	case "0", "f", "F", "false", "FALSE", "False", "NO", "no", "No", "N", "n", "OFF", "off", "Off":
		return false, nil
	}
	return false, errors.New("解析布尔值失败: " + val)
}
