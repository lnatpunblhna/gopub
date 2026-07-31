package confcontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/config"
	"github.com/linclin/gopub/src/library/jumpserver"
)

func ServerGroups(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}
	enableJumpserver, _ := config.Bool("enableJumpserver")
	if enableJumpserver {
		group2id, _ := jumpserver.GetGroups()
		return ctx.SetJson(0, group2id, "")
	}
	return ctx.SetJson(0, nil, "")
}
