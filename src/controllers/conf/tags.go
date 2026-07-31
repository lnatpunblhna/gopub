package confcontrollers

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/library/common"
	"github.com/linclin/gopub/src/library/db"
)

func Tags(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}

	rows, _ := db.Values("SELECT tag FROM `project`")

	var a []string
	for _, row := range rows {
		tmp := strings.Split(common.GetString(row["tag"]), " ")
		for _, tag := range tmp {
			if tag != "" {
				a = append(a, tag)
			}
		}
	}

	a = common.ArrayUnique(a)
	return ctx.SetJson(0, a, "")
}
