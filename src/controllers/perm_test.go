package controllers

import "testing"

// TestProjectAllowed 锁定权限矩阵。语义与 conf/mylist.go 的列表过滤一致：
// 列表里看得到的才能操作，未知角色一律无权限。
func TestProjectAllowed(t *testing.T) {
	cases := []struct {
		name  string
		role  int16
		level int16
		bound bool
		want  bool
	}{
		// Role 1：管理员，不看 level 也不看授权
		{"管理员-prod", roleAdmin, 3, false, true},
		{"管理员-simu", roleAdmin, 2, false, true},
		{"管理员-test", roleAdmin, 1, false, true},

		// Role 10：只允许预发布（level=2）
		{"预发布用户-simu", rolePreRelease, levelSimu, false, true},
		{"预发布用户-prod", rolePreRelease, 3, false, false},
		{"预发布用户-test", rolePreRelease, 1, false, false},
		{"预发布用户-即使有授权也不放行非 simu", rolePreRelease, 3, true, false},

		// Role 20：只认 group 表里的授权，与 level 无关
		{"单项目用户-已授权", roleSingle, 3, true, true},
		{"单项目用户-已授权-simu", roleSingle, levelSimu, true, true},
		{"单项目用户-未授权", roleSingle, levelSimu, false, false},

		// 未知角色：一律拒绝。LDAP 组没配 ldapGroupName2roleid_* 时会落到 0，
		// 收紧前这类账号在列表里能看到全部项目
		{"未知角色-0", 0, levelSimu, false, false},
		{"未知角色-0-即使有授权", 0, levelSimu, true, false},
		{"未知角色-30", 30, levelSimu, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := projectAllowed(c.role, c.level, c.bound); got != c.want {
				t.Errorf("projectAllowed(role=%d, level=%d, bound=%v) = %v, want %v",
					c.role, c.level, c.bound, got, c.want)
			}
		})
	}
}

// TestCanAccessProjectNilGuards 确认空值一律拒绝——未登录时 ctx.User 为 nil，
// 不能因此把项目挂到上下文上。
func TestCanAccessProjectNilGuards(t *testing.T) {
	if CanAccessProject(nil, nil) {
		t.Error("user 与 project 均为 nil 时应拒绝")
	}
	if CanAccessTask(nil, nil) {
		t.Error("task 为 nil 时应拒绝")
	}
}
