package p2pcontrollers

import (
	"encoding/json"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/components"
	"github.com/linclin/gopub/src/library/config"
	"github.com/linclin/gopub/src/library/p2p/init_sever"
	"github.com/linclin/gopub/src/models"
)

func Agent(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}

	s := components.BaseComponents{}
	s.SetProject(ctx.Project)
	s.SetTask(&models.Task{Id: -3})
	ips := s.GetHostIps()
	ss := init_sever.P2pSvc.CheckAllClient(ips)
	reIps := []string{}
	for ip, status := range ss {
		if status == "dead" {
			reIps = append(reIps, strings.Split(ip, ":")[0])
		}
	}
	if len(reIps) > 0 && ctx.Project.P2p == 1 {
		AgentDestDir := config.String("AgentDestDir")
		err := s.StartP2pAgent(reIps, AgentDestDir)
		if err != nil {
			return ctx.SetJson(1, nil, "重启失败"+err.Error())
		}
		return ctx.SetJson(0, nil, "重启成功")
	}
	return ctx.SetJson(0, nil, "已全部启动")
}

func AgentStart(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	ips := []string{}
	json.Unmarshal(ctx.RequestBody(), &ips)
	s := components.BaseComponents{}
	s.SetProject(ctx.Project)
	AgentDestDir := config.String("AgentDestDir")
	err := s.StartP2pAgent(ips, AgentDestDir)
	if err != nil {
		return ctx.SetJson(1, nil, "重启失败"+err.Error())
	}
	return ctx.SetJson(0, nil, "重启成功")
}
