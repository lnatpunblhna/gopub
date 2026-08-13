// 对象级权限：判断某个用户能不能碰某个项目 / 上线单。
//
// 收紧之前鉴权只有「登录 / 管理员」两级，projectId、taskId 都是照单全收地按 ID
// 查库，任何登录用户猜到 ID 就能读写别人的项目配置（其中含会在目标机执行的钩子脚本）。
//
// 这里的角色语义与 conf/mylist.go 的列表过滤保持一致，即"列表里看得到的，才能操作"：
//
//	Role 1  管理员         —— 全部项目
//	Role 10 全部预发布用户 —— 仅 level=2（simu）的项目
//	Role 20 单个项目用户   —— 仅 group 表里授权过的项目
//	其余值（含 LDAP 组未配 ldapGroupName2roleid_* 时落到的 0）—— 一律无权限
package controllers

import (
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
)

const (
	rolePreRelease int16 = 10 // 全部预发布用户
	roleSingle     int16 = 20 // 单个项目用户
	levelSimu      int16 = 2  // 预发布环境，与 components 里 getEnv 的 2=simu 对应
)

// projectAllowed 是权限矩阵的纯逻辑部分，抽出来便于单测。
// bound 表示 group 表里是否存在该 (user, project) 授权，仅对 Role 20 有意义。
func projectAllowed(role int16, level int16, bound bool) bool {
	switch role {
	case roleAdmin:
		return true
	case rolePreRelease:
		return level == levelSimu
	case roleSingle:
		return bound
	default:
		// 未知角色按最小权限处理，不再像旧的列表过滤那样默默放行全部
		return false
	}
}

// userBoundToProject 查 group 表里有没有该用户对该项目的授权
func userBoundToProject(userId int, projectId int) bool {
	rows, err := db.Values("SELECT id FROM `group` WHERE user_id = ? AND project_id = ? LIMIT 1", userId, projectId)
	if err != nil {
		// 查不动库时按拒绝处理，避免因为数据库抖动把权限判断放宽
		logger.Error("查询项目授权失败:", err)
		return false
	}
	return len(rows) > 0
}

// CanAccessProject 判断用户能否操作该项目。user 或 project 为空一律拒绝。
func CanAccessProject(user *models.User, project *models.Project) bool {
	if user == nil || user.Id == 0 || project == nil || project.Id == 0 {
		return false
	}
	bound := false
	if user.Role == roleSingle {
		bound = userBoundToProject(user.Id, project.Id)
	}
	return projectAllowed(user.Role, project.Level, bound)
}

// CanAccessProjectId 按项目 ID 判断，项目不存在时返回 false。
func CanAccessProjectId(user *models.User, projectId int) bool {
	if projectId == 0 {
		return false
	}
	project, err := models.GetProjectById(projectId)
	if err != nil || project == nil {
		return false
	}
	return CanAccessProject(user, project)
}

// CanAccessTask 判断用户能否操作该上线单，按其所属项目的权限决定。
func CanAccessTask(user *models.User, task *models.Task) bool {
	if task == nil || task.Id == 0 {
		return false
	}
	return CanAccessProjectId(user, task.ProjectId)
}
