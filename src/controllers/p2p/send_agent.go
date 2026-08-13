package p2pcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
)

func SendAgent(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}

	s := components.BaseComponents{}
	s.SetProject(ctx.Project)
	s.SetTask(&models.Task{})
	// 原来和 gitpull 共用 -3 这个占位 id，两类操作的日志混在一起
	s.SetScope(models.RecordScopeAgent)
	s.SetOperatorFromUser(ctx.User)
	agentDir := config.String("AgentDir")
	AgentDestDir := config.String("AgentDestDir")
	err := s.SendP2pAgent(agentDir, AgentDestDir)
	if err != nil {
		logger.Info("出错啦！")
		return ctx.SetJson(1, nil, "p2p文件传输失败，请检查配置，或目标机器权限"+err.Error())
	}
	return ctx.SetJson(0, nil, "更新agent成功")
}
