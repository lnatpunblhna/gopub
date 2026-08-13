package tasks

import (
	"github.com/lnatpunblhna/gopub/src/library/components"
	"github.com/lnatpunblhna/gopub/src/library/p2p/init_sever"
	"github.com/lnatpunblhna/gopub/src/models"
	"strings"
	"time"

	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
)

type emailConfig struct {
	UserName string `json:"username,omitempty"`
	PassWord string `json:"password,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
}

func init() {

}
func Check_p2p_angent_status() error {
	logger.Info("p2p agent自动任务开始" + time.Now().Format("2006-01-02 15:04:05"))
	var projects []models.Project
	num, err := db.QueryRows(&projects, "SELECT * FROM `project` WHERE  p2p=1")
	// 注意：此处条件沿用原实现（err != nil），看起来应为 err == nil，
	// 但该定时任务在 main 中已被注释停用，本次迁移不改动其判断逻辑。
	if num > 0 && err != nil {
		for _, project := range projects {
			s := components.BaseComponents{}
			s.SetProject(&project)
			s.SetTask(&models.Task{})
			s.SetScope(models.RecordScopeAgent)
			ips := s.GetHostIps()
			ss := init_sever.P2pSvc.CheckAllClient(ips)
			reIps := []string{}
			for ip, status := range ss {
				if status == "dead" {
					reIps = append(reIps, strings.Split(ip, ":")[0])
				}
			}
			if len(reIps) > 0 {
				AgentDestDir := config.String("AgentDestDir")
				s.StartP2pAgent(reIps, AgentDestDir)
			}
		}
	}
	logger.Info("p2p agent自动任务结束" + time.Now().Format("2006-01-02 15:04:05"))
	return nil
}
