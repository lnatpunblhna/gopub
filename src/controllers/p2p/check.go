package p2pcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/library/p2p/init_sever"
	"github.com/lnatpunblhna/gopub/src/models"
)

type P2pinfo struct {
	Host   string
	Status string
	Pid    int
	Pname  string
}

func Check(c echo.Context) error {
	ctx := controllers.New(c)
	searchtype := ctx.GetString("type")
	projectId := ctx.GetString("projectId")
	logger.Info(searchtype)

	if searchtype == "0" {
		var projects []models.Project
		var p []P2pinfo
		ss := map[string]string{}
		i, err := db.QueryRows(&projects, "SELECT * FROM `project` WHERE `p2p` = 1 ")
		if i <= 0 || err != nil {
			return ctx.SetJson(1, ss, "no agent")
		}
		for _, project := range projects {
			s := components.BaseComponents{}
			s.SetProject(&project)
			ips := s.GetHostIps()
			proRes := init_sever.P2pSvc.CheckAllClient(ips)
			for key, value := range proRes {
				if value == "dead" {
					if !common.InList(key, ss) {
						ss[key] = value
						p = append(p, P2pinfo{
							Host:   key,
							Status: value,
							Pid:    project.Id,
							Pname:  project.Name,
						})
					}
				}
			}
		}
		logger.Info(p)
		return ctx.SetJson(0, p, "")
	}

	if projectId != "" && searchtype == "1" {
		var projects []models.Project
		ss := map[string]string{}
		i, err := db.QueryRows(&projects, "SELECT * FROM `project` WHERE `id` = ?   ", projectId)
		if i <= 0 || err != nil {
			return ctx.SetJson(1, ss, "no agent")
		}
		for _, project := range projects {
			s := components.BaseComponents{}
			s.SetProject(&project)
			ips := s.GetHostIps()
			proRes := init_sever.P2pSvc.CheckAllClient(ips)
			for key, value := range proRes {
				if !common.InList(key, ss) {
					ss[key] = value
				}
			}
		}
		return ctx.SetJson(0, ss, "")
	}
	return nil
}
