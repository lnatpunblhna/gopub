package confcontrollers

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/models"
)

func Copy(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}
	if ctx.Project == nil || ctx.Project.Id == 0 {
		return ctx.SetJson(1, nil, "Parameter error")
	}
	ctx.Project.Name = ctx.Project.Name + " - copy"
	ctx.Project.Id = 0
	ctx.Project.UserId = uint(ctx.User.Id)
	now := time.Now()
	ctx.Project.CreatedAt = now
	ctx.Project.UpdatedAt = now
	_, err := models.AddProject(ctx.Project)
	if err != nil {
		return ctx.SetJson(1, nil, "复制失败")
	}
	return ctx.SetJson(0, ctx.Project, "")
}
