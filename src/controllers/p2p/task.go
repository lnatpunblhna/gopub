package p2pcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/p2p/init_sever"
)

func Task(c echo.Context) error {
	ctx := controllers.New(c)
	taskId := ctx.GetString("taskId")
	ss, _ := init_sever.P2pSvc.QueryTaskNoHttp(taskId)
	return ctx.SetJson(0, ss, "")
}
