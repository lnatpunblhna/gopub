package usercontrollers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/db"
)

func UserProject(c echo.Context) error {
	ctx := controllers.New(c)
	userId := ctx.GetString("user_id")
	projects, _ := db.Values("SELECT  project.id,project.`name`,project.`level` FROM `group` left JOIN project on  group.project_id=project.id WHERE `group`.user_id= ?", userId)
	return ctx.SetJson(0, projects, "")
}
