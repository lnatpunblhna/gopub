package components

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lnatpunblhna/gopub/src/library/logger"
	gopubssh "github.com/lnatpunblhna/gopub/src/library/ssh"
	"github.com/lnatpunblhna/gopub/src/models"
)

// 这里定义写进 record.memo 的数据结构。
//
// 旧实现直接 json.Marshal(ExecResult)，而它的 Error 是 error 接口，
// 序列化之后只剩 `{}`，前端读的 ErrorInfo 字段更是压根不存在，
// 于是「命令为什么失败」在页面上完全看不到。现在改成显式字段。

// HostResult 是单台机器（本地命令则是宿主机）的执行结果
type HostResult struct {
	Host      string `json:"host"`
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ErrorInfo string `json:"errorInfo"`
	ExitCode  int    `json:"exitCode"`
	StartTime int64  `json:"startTime"`
	Duration  int    `json:"duration"`
	Truncated bool   `json:"truncated"`
}

// StepRecord 是一条 record 的 memo 内容
type StepRecord struct {
	Kind      string       `json:"kind"`              // local / remote / transfer / final
	Success   bool         `json:"success"`           // 整步是否成功
	Truncated bool         `json:"truncated"`         // 是否发生过截断
	LogFile   string       `json:"logFile,omitempty"` // 完整日志的相对路径，供下载接口使用
	Note      string       `json:"note,omitempty"`
	Hosts     []HostResult `json:"hosts"`
}

const (
	// 单台机器单路输出（stdout / stderr）入库上限
	perStreamLimit = 8 << 10
	// 整条 memo 的入库上限。memo 若仍是 TEXT，MySQL 的硬上限是 65535 字节，
	// 超了整条 UPDATE 直接失败 —— 那一步在页面上就是一片空白，
	// 所以这里留足余量，宁可截断也不能让写入失败。
	memoLimit = 56 << 10
)

// TaskLogRoot 是完整日志的落盘根目录。定时清理任务也按这个路径回收文件，
// 所以导出而不是各处各写一份字面量。
const TaskLogRoot = "logs/task_log"

// SetScope 指定这批记录属于哪类操作（上线 / 检测 / 刷新 ...）
func (c *BaseComponents) SetScope(scope string) {
	c.scope = scope
}

// SetStage 指定后续记录属于上线流程的第几步（10/20/.../60，与前端步骤条对应）。
//
// 旧实现是每走完一步执行一次
// `UPDATE record SET action=? WHERE task_id=? AND action=0` 事后回填，
// 既要在无索引的表上全表扫描，又让增量拉取拿不到已取走记录的更新。
// 记录诞生时就带上自己的阶段，两个问题一起消失。
func (c *BaseComponents) SetStage(stage uint) {
	c.stage = stage
}

// SetAttempt 指定这批记录属于第几次发布尝试
func (c *BaseComponents) SetAttempt(attempt int) {
	c.attempt = attempt
}

func (c *BaseComponents) recordAttempt() int {
	if c.attempt <= 0 {
		return 1
	}
	return c.attempt
}

// SetOperator 指定当前操作人。检测 / 刷新 / gitpull 这些没有上线单的操作
// 靠它把不同人的日志分开，避免几个人同时点检测时输出串在一起。
func (c *BaseComponents) SetOperator(userId uint) {
	c.operatorId = userId
}

// SetOperatorFromUser 是 SetOperator 的便捷写法，user 为空时不改动
func (c *BaseComponents) SetOperatorFromUser(user *models.User) {
	if user != nil && user.Id > 0 {
		c.operatorId = uint(user.Id)
	}
}

// SaveRecordNote 记录一条非命令类的结果（例如 p2p agent 存活检测）
func (c *BaseComponents) SaveRecordNote(id int, success bool, detail string) {
	if id <= 0 {
		return
	}
	re, err := models.GetRecordById(id)
	if err != nil {
		logger.Error("读取上线记录失败:", err)
		return
	}
	step := &StepRecord{Kind: "local", Success: success, Hosts: []HostResult{{
		Host:      "-",
		Stdout:    detail,
		StartTime: time.Now().Unix(),
	}}}
	if !success {
		step.Hosts[0].ErrorInfo = detail
		step.Hosts[0].Stdout = ""
		step.Hosts[0].ExitCode = -1
	}
	truncateStep(step)
	re.Status = 1
	if !success {
		re.Status = 0
	}
	re.Memo = marshalMemo(step)
	if err := models.UpdateRecordById(re); err != nil {
		logger.Error("回填上线记录失败:", err)
	}
}

// DisableRecord 关闭记录写入。
// 列分支 / tag / commit、版本对比这类只读查询原本也在往 record 里写，
// 它们的错误本来就会通过接口返回给前端，写库只会把上线日志冲得乱七八糟，
// 并且让表无节制地膨胀（task.Id 为 0 时全落到 task_id=-99）。
func (c *BaseComponents) DisableRecord() {
	c.recordOff = true
}

func (c *BaseComponents) recordScope() string {
	if c.scope == "" {
		return models.RecordScopeRelease
	}
	return c.scope
}

func (c *BaseComponents) recordTaskId() int64 {
	if c.task == nil || c.task.Id == 0 {
		return 0
	}
	return int64(c.task.Id)
}

func (c *BaseComponents) recordUserId() uint {
	if c.operatorId > 0 {
		return c.operatorId
	}
	if c.task != nil {
		return c.task.UserId
	}
	return 0
}

func (c *BaseComponents) recordProjectId() int {
	if c.project == nil {
		return 0
	}
	return c.project.Id
}

// SaveRecord 在命令开始执行时先落一条记录，返回其 id；关闭记录时返回 0。
//
// created_at 在这里就写入。旧实现留到执行结束才补，于是「执行中」的记录
// created_at 为 0，按时间过滤的页面上根本看不到正在跑的命令。
func (c *BaseComponents) SaveRecord(command string) int {
	if c.recordOff {
		return 0
	}
	re := models.Record{
		Command:   command,
		TaskId:    c.recordTaskId(),
		UserId:    c.recordUserId(),
		ProjectId: c.recordProjectId(),
		Scope:     c.recordScope(),
		Attempt:   c.recordAttempt(),
		Action:    c.stage,
		Status:    1,
		CreatedAt: int(time.Now().Unix()),
	}
	if re.TaskId == 0 {
		// 没有上线单的操作按 scope 归到固定的占位 taskId，与前端约定一致
		re.TaskId = pseudoTaskIdOf(re.Scope)
	}
	id, err := models.AddRecord(&re)
	if err != nil {
		logger.Error("写入上线记录失败:", err)
		return 0
	}
	return int(id)
}

// pseudoTaskIdOf 把 scope 映射回前端使用的占位 taskId
func pseudoTaskIdOf(scope string) int64 {
	switch scope {
	case models.RecordScopeDetect:
		return models.PseudoTaskDetect
	case models.RecordScopeFlush:
		return models.PseudoTaskFlush
	case models.RecordScopeGitPull:
		return models.PseudoTaskGitPull
	case models.RecordScopeAgent:
		return models.PseudoTaskAgent
	}
	return models.PseudoTaskUnknown
}

// SaveRecordRes 回填执行结果：完整日志落盘，截断后的摘要入库。
func (c *BaseComponents) SaveRecordRes(id int, kind string, results []gopubssh.ExecResult, stepErr error) {
	if id <= 0 {
		return
	}
	re, err := models.GetRecordById(id)
	if err != nil {
		logger.Error("读取上线记录失败:", err)
		return
	}

	step := buildStepRecord(kind, results, stepErr)
	// 先落盘再入库：memo 里要带上完整日志的路径
	step.LogFile = c.writeStepLog(re, step)
	truncateStep(step)

	re.Status = 1
	if !step.Success {
		re.Status = 0
	}
	re.Duration = stepDuration(results)
	if started := stepStartedAt(results); started > 0 {
		re.CreatedAt = started
	}
	re.Memo = marshalMemo(step)
	if err := models.UpdateRecordById(re); err != nil {
		logger.Error("回填上线记录失败:", err)
	}
}

// AddFinalRecord 写一条终结记录，标记整个流程成功或失败。
//
// 旧实现里失败只改 task 状态，record 里没有任何痕迹：
// 前端既拿不到失败原因，也不知道流程已经结束，会一直空转轮询。
func (c *BaseComponents) AddFinalRecord(success bool, stage string, err error) {
	if c.recordOff {
		return
	}
	command := "===== 发布完成 ====="
	status := int16(1)
	if !success {
		command = "===== 发布失败：" + stage + " ====="
		status = 0
	}
	step := &StepRecord{Kind: "final", Success: success, Hosts: []HostResult{}}
	if err != nil {
		step.Hosts = []HostResult{{
			Host:      "-",
			ErrorInfo: err.Error(),
			ExitCode:  -1,
			StartTime: time.Now().Unix(),
		}}
	}
	truncateStep(step)
	re := models.Record{
		Command:   command,
		TaskId:    c.recordTaskId(),
		UserId:    c.recordUserId(),
		ProjectId: c.recordProjectId(),
		Scope:     c.recordScope(),
		Attempt:   c.recordAttempt(),
		Status:    status,
		// 前端据此判定流程已结束，停止轮询
		Action:    actionFinished,
		Memo:      marshalMemo(step),
		CreatedAt: int(time.Now().Unix()),
	}
	if re.TaskId == 0 {
		re.TaskId = pseudoTaskIdOf(re.Scope)
	}
	if _, err := models.AddRecord(&re); err != nil {
		logger.Error("写入终结记录失败:", err)
	}
}

// actionFinished 是终结记录的 action 取值，与流程中的 10/20/.../60 区分开
const actionFinished = 100

// buildStepRecord 把执行结果转成入库结构
func buildStepRecord(kind string, results []gopubssh.ExecResult, stepErr error) *StepRecord {
	step := &StepRecord{Kind: kind, Success: stepErr == nil, Hosts: make([]HostResult, 0, len(results))}
	for _, r := range results {
		host := r.Host
		if host == "" {
			host = "localhost"
		}
		h := HostResult{
			Host:      host,
			Stdout:    r.Stdout,
			Stderr:    r.Stderr,
			ErrorInfo: r.ErrorInfo(),
			ExitCode:  r.ExitCode,
		}
		if r.LocalFilePath != "" || r.RemoteFilePath != "" {
			h.Stdout = strings.TrimRight(h.Stdout+"\n"+r.LocalFilePath+" -> "+r.RemoteFilePath, "\n")
		}
		if !r.StartTime.IsZero() {
			h.StartTime = r.StartTime.Unix()
			if !r.EndTime.IsZero() && r.EndTime.After(r.StartTime) {
				h.Duration = int(r.EndTime.Sub(r.StartTime).Seconds())
			}
		}
		step.Hosts = append(step.Hosts, h)
	}
	// 单机执行失败但 ExecResult 里没带上错误时（例如上层直接返回 error），
	// 至少把整步的错误挂上去，不能让页面上什么都没有
	if stepErr != nil && !anyHostFailed(step.Hosts) {
		step.Hosts = append(step.Hosts, HostResult{
			Host:      "-",
			ErrorInfo: stepErr.Error(),
			ExitCode:  -1,
			StartTime: time.Now().Unix(),
		})
	}
	return step
}

func anyHostFailed(hosts []HostResult) bool {
	for _, h := range hosts {
		if h.ErrorInfo != "" {
			return true
		}
	}
	return false
}

// stepDuration 取各主机里最长的耗时作为整步耗时
func stepDuration(results []gopubssh.ExecResult) int {
	max := 0
	for _, r := range results {
		if r.StartTime.IsZero() || r.EndTime.IsZero() || !r.EndTime.After(r.StartTime) {
			continue
		}
		if d := int(r.EndTime.Sub(r.StartTime).Seconds()); d > max {
			max = d
		}
	}
	return max
}

// stepStartedAt 取最早的开始时间；全部无效时返回 0，由调用方保留入库时写的时间。
// 这里必须挡住零值 time.Time —— 它的 Unix() 是 -62135596800，
// 一旦写进 created_at，这条记录在按时间过滤的页面上就再也不会出现。
func stepStartedAt(results []gopubssh.ExecResult) int {
	earliest := 0
	for _, r := range results {
		if r.StartTime.IsZero() {
			continue
		}
		ts := int(r.StartTime.Unix())
		if ts <= 0 {
			continue
		}
		if earliest == 0 || ts < earliest {
			earliest = ts
		}
	}
	return earliest
}

// truncateStep 按上限裁剪输出，保证 memo 一定写得进去
func truncateStep(step *StepRecord) {
	for i := range step.Hosts {
		var cut bool
		step.Hosts[i].Stdout, cut = truncateText(step.Hosts[i].Stdout, perStreamLimit)
		step.Hosts[i].Truncated = step.Hosts[i].Truncated || cut
		step.Hosts[i].Stderr, cut = truncateText(step.Hosts[i].Stderr, perStreamLimit)
		step.Hosts[i].Truncated = step.Hosts[i].Truncated || cut
		step.Hosts[i].ErrorInfo, cut = truncateText(step.Hosts[i].ErrorInfo, perStreamLimit)
		step.Hosts[i].Truncated = step.Hosts[i].Truncated || cut
		if step.Hosts[i].Truncated {
			step.Truncated = true
		}
	}
	if len(marshalMemo(step)) <= memoLimit {
		return
	}
	// 还是太大（几十台机器同时刷屏）：只留失败主机的输出，成功的收敛成一行
	step.Truncated = true
	step.Note = "输出过大，成功主机的内容已省略，完整日志请下载"
	for i := range step.Hosts {
		if step.Hosts[i].ErrorInfo != "" {
			continue
		}
		step.Hosts[i].Stdout = "(已省略，完整日志请下载)"
		step.Hosts[i].Stderr = ""
	}
	if len(marshalMemo(step)) <= memoLimit {
		return
	}
	// 极端情况：失败主机本身就有海量输出，逐台再压一次
	for i := range step.Hosts {
		step.Hosts[i].Stdout, _ = truncateText(step.Hosts[i].Stdout, 1<<10)
		step.Hosts[i].Stderr, _ = truncateText(step.Hosts[i].Stderr, 1<<10)
		step.Hosts[i].ErrorInfo, _ = truncateText(step.Hosts[i].ErrorInfo, 1<<10)
	}
}

// truncateText 保留首尾两段，中间标注省略了多少字节
func truncateText(s string, limit int) (string, bool) {
	if len(s) <= limit {
		return s, false
	}
	head := limit / 2
	tail := limit - head
	omitted := len(s) - limit
	return s[:head] +
		fmt.Sprintf("\n... [已省略 %d 字节，完整日志请下载] ...\n", omitted) +
		s[len(s)-tail:], true
}

func marshalMemo(step *StepRecord) string {
	b, err := json.Marshal(step)
	if err != nil {
		logger.Error("序列化上线记录失败:", err)
		return `{"kind":"local","success":false,"hosts":[]}`
	}
	return string(b)
}

// writeStepLog 把未截断的完整输出写到文件，返回相对 TaskLogRoot 的路径。
// 这是 LogTaskCommond 那段被注释掉的文件日志的替代实现：
// 入库的内容一定是截断过的，真正排查问题时需要能拿到全文。
func (c *BaseComponents) writeStepLog(re *models.Record, step *StepRecord) string {
	rel := filepath.Join(fmt.Sprintf("task-%d", re.TaskId), fmt.Sprintf("%d.log", re.Id))
	full := filepath.Join(TaskLogRoot, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		logger.Error("创建上线日志目录失败:", err)
		return ""
	}

	var b strings.Builder
	b.WriteString("# " + time.Unix(int64(re.CreatedAt), 0).Format("2006-01-02 15:04:05") + "\n")
	b.WriteString("# command\n" + re.Command + "\n")
	for _, h := range step.Hosts {
		b.WriteString(fmt.Sprintf("\n===== host: %s (exit=%d, %ds) =====\n", h.Host, h.ExitCode, h.Duration))
		if h.Stdout != "" {
			b.WriteString("--- stdout ---\n" + h.Stdout + "\n")
		}
		if h.Stderr != "" {
			b.WriteString("--- stderr ---\n" + h.Stderr + "\n")
		}
		if h.ErrorInfo != "" {
			b.WriteString("--- error ---\n" + h.ErrorInfo + "\n")
		}
	}
	if err := os.WriteFile(full, []byte(b.String()), 0o644); err != nil {
		logger.Error("写入上线日志文件失败:", err)
		return ""
	}
	return rel
}

// TaskLogPath 依据 memo 里记录的相对路径还原出磁盘路径，供下载接口使用。
// 路径来自数据库，必须限制在日志根目录内，防止被构造成目录穿越。
func TaskLogPath(rel string) (string, bool) {
	if rel == "" {
		return "", false
	}
	root, err := filepath.Abs(TaskLogRoot)
	if err != nil {
		return "", false
	}
	full, err := filepath.Abs(filepath.Join(root, filepath.Clean("/"+rel)))
	if err != nil {
		return "", false
	}
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", false
	}
	if st, err := os.Stat(full); err != nil || st.IsDir() {
		return "", false
	}
	return full, true
}

// LogFileOfMemo 从 memo 里取出完整日志的相对路径
func LogFileOfMemo(memo string) string {
	if memo == "" {
		return ""
	}
	var step StepRecord
	if err := json.Unmarshal([]byte(memo), &step); err != nil {
		return ""
	}
	return step.LogFile
}
