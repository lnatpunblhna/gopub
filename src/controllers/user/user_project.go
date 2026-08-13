package usercontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/db"
)

func UserProject(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}
	userId := ctx.GetString("user_id")
	// 只有管理员（Role==1，与 changepasswd 等处的判断一致）能查别人的项目授权，
	// 其余账号一律只能看自己的
	if ctx.User.Role != 1 {
		userId = common.GetString(ctx.User.Id)
	}
	projects, _ := db.Values("SELECT  project.id,project.`name`,project.`level` FROM `group` left JOIN project on  group.project_id=project.id WHERE `group`.user_id= ?", userId)
	return ctx.SetJson(0, projects, "")
}
