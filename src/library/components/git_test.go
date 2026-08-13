package components

import "testing"

// parseRemoteHeadBranch 取代的是原先内联在 && 链里的
// BR=$(git remote show origin | sed -n 's/.*HEAD branch: //p')。
// 那个写法拿不到分支时会静默得到空串,一路拼到 git checkout -q "" 才炸成
// fatal: empty string is not a valid pathspec。这里把"拿不到"的各种形态都钉住,
// 确保它们返回空串,由调用方 resolveBranch 转成可读错误。
func TestParseRemoteHeadBranch(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "常规输出",
			in: `* remote origin
  Fetch URL: git@example.com:foo/bar.git
  Push  URL: git@example.com:foo/bar.git
  HEAD branch: master
  Remote branches:
    master tracked`,
			want: "master",
		},
		{
			name: "默认分支是 main",
			in:   "  HEAD branch: main\n",
			want: "main",
		},
		{
			name: "带斜杠的分支名",
			in:   "  HEAD branch: release/test\n",
			want: "release/test",
		},
		// 远程没设置 HEAD 时 git 打印的就是这个字面量,它不是分支名
		{"HEAD 未知", "  HEAD branch: (unknown)\n", ""},
		{"冒号后为空", "  HEAD branch:\n", ""},
		// git remote show origin 失败时 stdout 为空,原实现在这里得到空 BR 却照样往下走
		{"空输出", "", ""},
		// 输出被本地化的情况:匹配不到就得当探测失败,不能返回垃圾值
		{"非英文 locale", "  头部分支：master\n", ""},
		{"认证失败只有 stderr", "fatal: Could not read from remote repository.\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseRemoteHeadBranch(c.in); got != c.want {
				t.Errorf("parseRemoteHeadBranch() = %q, want %q", got, c.want)
			}
		})
	}
}
