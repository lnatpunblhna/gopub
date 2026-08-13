package common

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// 发布流程里的本地命令最终都会走 exec.Command("/bin/bash", "-c", cmd)
// （见 src/library/ssh/mainfunc.go），远端命令也是整串交给 sshd 执行。
// 也就是说凡是拼进命令串的外部输入都会被 shell 解释，必须先过这里。
//
// 两道防线配合使用：
//   - ShellQuote 负责让内容无论如何都只被当成一个字面量参数；
//   - ValidGitRef / ValidHTTPURL 负责在入口就挡掉明显不合法的取值，
//     避免把奇怪的东西塞给 git / wget 后产生难以排查的报错。

// gitRefPattern 覆盖 git 允许的分支 / tag / commit 写法：
// 字母数字加 . _ / - 以及 ^ ~ @ { } 这些常见的修订限定符。
// 刻意不含空格、引号、$、` 、; 、& 、| 、换行等 shell 元字符。
var gitRefPattern = regexp.MustCompile(`^[A-Za-z0-9._/@^~{}-]+$`)

// ShellQuote 把任意字符串包成单引号字面量，使其在 shell 里只被当作一个参数。
// 单引号内除了单引号本身没有任何特殊字符，所以只需把内部的 ' 拆开重接：
// abc'def -> 'abc'\”def'
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ValidGitRef 校验分支 / tag / commit 是否是安全且合法的 git 引用写法。
// 空串交给调用方决定语义（分支为空表示"用远程默认分支"），这里不作判断。
func ValidGitRef(ref string) error {
	if ref == "" {
		return nil
	}
	if len(ref) > 255 {
		return errors.New("git 引用过长")
	}
	if !gitRefPattern.MatchString(ref) {
		return errors.New("git 引用含非法字符，只允许字母、数字与 . _ / - @ ^ ~ { }")
	}
	// git 自身也拒绝这些写法，提前挡掉可以给出更清楚的提示
	if strings.HasPrefix(ref, "-") {
		return errors.New("git 引用不能以 - 开头")
	}
	if strings.Contains(ref, "..") {
		return errors.New("git 引用不能包含 ..")
	}
	return nil
}

// relPathPattern 是制品相对路径允许的写法：字母数字加 . _ / - 以及 %（转义字符）。
var relPathPattern = regexp.MustCompile(`^[A-Za-z0-9._/%-]+$`)

// ValidDownloadRef 校验制品地址。它有两种形态（见 components.BaseFile.UpdateRepo）：
// 以 http 开头时是完整下载地址；否则是相对路径，会被拼到项目的 repo_url 后面。
func ValidDownloadRef(raw string) error {
	if raw == "" {
		return nil
	}
	if len(raw) > 2048 {
		return errors.New("制品地址过长")
	}
	if strings.ContainsAny(raw, " \t\r\n") {
		return errors.New("制品地址不能包含空白字符")
	}

	// 与 UpdateRepo / CheckFiles 的判断保持一致：只看是不是 http 前缀
	if !strings.HasPrefix(raw, "http") {
		if !relPathPattern.MatchString(raw) {
			return errors.New("制品相对路径含非法字符，只允许字母、数字与 . _ / - %")
		}
		if strings.Contains(raw, "..") {
			return errors.New("制品相对路径不能包含 ..")
		}
		return nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("制品地址格式错误：" + err.Error())
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("制品地址只支持 http / https")
	}
	if u.Host == "" {
		return errors.New("制品地址缺少主机名")
	}
	return nil
}
