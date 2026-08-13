package gopubssh

import (
	"os/exec"
	"time"

	"golang.org/x/crypto/ssh"
)

// ExecResult 取代原先直接复用的 sshexec.ExecResult。
//
// 换掉它的原因：sshexec.ExecResult 只有一个 `Error error` 字段，
// 而 error 是接口，json.Marshal 之后只剩 `{}`（errors.errorString 没有导出字段），
// 于是发布日志入库后错误原因全部丢失，前端只能显示一片空白。
// 这里把错误摊平成 ErrorInfo/ExitCode 字符串与整数，并把 stdout / stderr 分开保存，
// 保证「命令为什么失败」这件事一定能落到 record 里。
//
// Result 字段的语义与旧版保持一致（成功取 stdout，失败取 stderr），
// 因为 components 层多处解析它的文本输出（如 git 分支列表）。
type ExecResult struct {
	Id             int
	Host           string
	Command        string
	LocalFilePath  string
	RemoteFilePath string
	Result         string
	Stdout         string
	Stderr         string
	ExitCode       int
	StartTime      time.Time
	EndTime        time.Time
	Error          error
}

// ErrorInfo 返回可读的错误文本，无错误时为空串。
func (r ExecResult) ErrorInfo() string {
	if r.Error == nil {
		return ""
	}
	return r.Error.Error()
}

// fail 统一填充失败信息：记录错误、退出码与结束时间。
// 退出码取不到（连接失败、超时等非命令退出的场景）时记 -1。
func (r *ExecResult) fail(err error) {
	r.Error = err
	r.ExitCode = exitCodeOf(err)
	if r.EndTime.IsZero() {
		r.EndTime = time.Now()
	}
}

// exitCodeOf 从本地 / 远端执行错误里提取退出码
func exitCodeOf(err error) int {
	if err == nil {
		return 0
	}
	switch e := err.(type) {
	case *exec.ExitError:
		return e.ExitCode()
	case *ssh.ExitError:
		return e.ExitStatus()
	}
	return -1
}
