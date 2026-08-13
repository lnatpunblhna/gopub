package confcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/jumpserver"
	"github.com/lnatpunblhna/gopub/src/library/logger"
)

func ServerGroups(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}
	enableJumpserver, _ := config.Bool("enableJumpserver")
	if enableJumpserver {
		group2id, err := jumpserver.GetGroups()
		if err != nil {
			logger.Error("获取 jumpserver 服务器分组失败:", err)
			return ctx.SetJson(1, nil, "获取服务器分组失败："+err.Error())
		}
		return ctx.SetJson(0, group2id, "")
	}
	return ctx.SetJson(0, nil, "")
}
