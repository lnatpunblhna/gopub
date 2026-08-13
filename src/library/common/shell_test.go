package common

import (
	"os/exec"
	"strings"
	"testing"
)

func TestShellQuote(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"普通字符串", "master", "'master'"},
		{"含空格", "a b", "'a b'"},
		{"含单引号", "abc'def", `'abc'\''def'`},
		{"分号注入", "a;whoami", "'a;whoami'"},
		{"命令替换", "$(whoami)", "'$(whoami)'"},
		{"反引号", "`whoami`", "'`whoami`'"},
		{"空串", "", "''"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ShellQuote(c.in); got != c.want {
				t.Errorf("ShellQuote(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestShellQuoteNeutralisesInjection 把引用后的串真的交给 bash 跑一遍，
// 确认注入载荷只会被当成字面量回显，不会产生额外的命令执行。
func TestShellQuoteNeutralisesInjection(t *testing.T) {
	payloads := []string{
		"a;echo INJECTED",
		"$(echo INJECTED)",
		"`echo INJECTED`",
		"a';echo INJECTED;'",
		"a && echo INJECTED",
		"a | echo INJECTED",
	}
	for _, p := range payloads {
		t.Run(p, func(t *testing.T) {
			out, err := exec.Command("/bin/bash", "-c", "echo "+ShellQuote(p)).Output()
			if err != nil {
				t.Fatalf("执行失败: %v", err)
			}
			got := strings.TrimRight(string(out), "\n")
			if got != p {
				t.Errorf("载荷被 shell 解释了: echo %s 输出 %q, want %q", ShellQuote(p), got, p)
			}
		})
	}
}

// TestShellQuoteInAssignment 复刻 components/git.go 里 `BR=<分支>` 的写法。
// 这是修复前真正被利用的位置：checkout 那一步虽然写了 "$BR"，
// 但赋值语句本身没有引用，分号后面的内容会被当成独立命令执行。
func TestShellQuoteInAssignment(t *testing.T) {
	payload := "a;echo INJECTED"
	// 未引用：注入生效（这就是修复前的行为，作为对照）
	raw, err := exec.Command("/bin/bash", "-c", "BR="+payload+" && echo \"[$BR]\"").Output()
	if err != nil {
		t.Fatalf("对照组执行失败: %v", err)
	}
	if !strings.Contains(string(raw), "INJECTED") {
		t.Fatalf("对照组未复现注入，测试前提不成立: %q", raw)
	}

	// 引用后：整串只是一个字面量赋值
	out, err := exec.Command("/bin/bash", "-c", "BR="+ShellQuote(payload)+" && echo \"[$BR]\"").Output()
	if err != nil {
		t.Fatalf("执行失败: %v", err)
	}
	got := strings.TrimRight(string(out), "\n")
	if got != "["+payload+"]" {
		t.Errorf("引用后仍被 shell 解释: 输出 %q, want %q", got, "["+payload+"]")
	}
}

func TestValidGitRef(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"空串放行", "", false},
		{"普通分支", "master", false},
		{"带斜杠", "feature/login", false},
		{"带点号", "release-1.2.3", false},
		{"commit id", "a1b2c3d4", false},
		{"修订限定符", "HEAD~2", false},
		{"分号注入", "a;whoami", true},
		{"命令替换", "$(whoami)", true},
		{"反引号", "`whoami`", true},
		{"含空格", "a b", true},
		{"管道", "a|b", true},
		{"以横线开头", "-oProxyCommand=x", true},
		{"含两点", "a..b", true},
		{"单引号", "a'b", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidGitRef(c.in)
			if (err != nil) != c.wantErr {
				t.Errorf("ValidGitRef(%q) err = %v, wantErr = %v", c.in, err, c.wantErr)
			}
		})
	}
}

func TestValidDownloadRef(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"空串放行", "", false},
		{"http 地址", "http://example.com/a.tar.gz", false},
		{"https 地址", "https://example.com/a.tar.gz", false},
		{"相对路径", "dist/app.tar.gz", false},
		{"相对路径带转义", "dist/app%20v1.tar.gz", false},
		{"缺主机名", "http://", true},
		{"file 协议", "file:///etc/passwd", true},
		{"相对路径注入", "a.tar.gz;whoami", true},
		{"相对路径命令替换", "$(whoami)", true},
		{"相对路径穿越", "../../etc/passwd", true},
		{"含空格", "http://example.com/a b", true},
		{"含换行", "http://example.com/a\nb", true},
		{"单引号逃逸", "http://x';whoami;echo '", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidDownloadRef(c.in)
			if (err != nil) != c.wantErr {
				t.Errorf("ValidDownloadRef(%q) err = %v, wantErr = %v", c.in, err, c.wantErr)
			}
		})
	}
}
