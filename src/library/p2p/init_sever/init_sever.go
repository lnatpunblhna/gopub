package init_sever

import (
	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/library/p2p/common"
	"github.com/lnatpunblhna/gopub/src/library/p2p/server"
	"os"
)

var P2pSvc *server.Server

func init() {

}
func Start() {
	cfg := common.ReadJson("agent/server.json")
	_, err := common.ParserConfig(&cfg)
	cfg.Server = true
	P2pSvc, err = server.NewServer(&cfg)
	if err != nil {
		logger.Error("start server error, %s.\n", err.Error())
		if config.RunMode() != "docker" {
			os.Exit(4)
		}
	}
	logger.Info("服务端p2p配置检测成功")
	if err := P2pSvc.Start(); err != nil {
		logger.Error("Start service failed, %s.\n", err.Error())
		if config.RunMode() != "docker" {
			os.Exit(4)
		}
	}
}
