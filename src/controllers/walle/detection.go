package wallecontrollers

import (
	"os"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/library/p2p/init_sever"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Detection(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	s := components.BaseComponents{}
	s.SetProject(ctx.Project)
	s.SetTask(&models.Task{})
	// 检测没有上线单，按 scope + 操作人归档，避免多人同时检测时日志串在一起
	s.SetScope(models.RecordScopeDetect)
	s.SetOperatorFromUser(ctx.User)
	codeBaseDir := s.GetDeployFromDir()
	//1:本地文件权限加成
	if _, err := os.Stat(codeBaseDir); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(codeBaseDir, os.ModePerm)
		}
	} else {
		_, err := os.Create(codeBaseDir + "/test.log")
		if err != nil {
			return ctx.SetJson(1, nil, "本地文件权限错误"+err.Error())
		}
		os.Remove(codeBaseDir + "/test.log")
	}

	if ctx.Project.RepoType == "git" {
		//2:git信任是否添加
		g := components.BaseGit{}
		g.SetBaseComponents(s)

		err := g.UpdateRepo("", "")
		if err != nil {
			//清空后再试一次 要是不行在退出
			err := s.RemoveLocalProjectWorkspace()
			if err != nil {
				return ctx.SetJson(1, nil, "清空目录错误"+err.Error())
			}
			err = g.UpdateRepo("", "")
			if err != nil {
				return ctx.SetJson(1, nil, "git拉取代码错误"+err.Error())
			}
		}
	}
	// 3.权限与免密码登录检测
	err := s.TestSsh()
	if err != nil {
		return ctx.SetJson(1, nil, "ssh目标机器错误"+err.Error())
	}
	// 4.检测用户是否具有目标机release目录读写权限
	err = s.TestReleaseDir()
	if err != nil {
		return ctx.SetJson(1, nil, "用户不具有目标机release目录读写权限"+err.Error())
	}
	//5推送p2p客户端并启动服务
	if ctx.Project.P2p == 1 {
		//这里做alive检测
		ips := s.GetHostIps()
		rid := s.SaveRecord("check p2p agent")
		ss := init_sever.P2pSvc.CheckAllClient(ips)
		detail := ""
		dead := false
		for ip, status := range ss {
			detail += ip + ": " + status + "\n"
			if status == "dead" {
				dead = true
			}
		}
		if dead {
			s.SaveRecordNote(rid, false, "p2p agent 未启动\n"+detail)
			return ctx.SetJson(1, nil, "p2p agent 未启动")
		}
		s.SaveRecordNote(rid, true, detail)
	}
	return ctx.SetJson(0, nil, "")
}
