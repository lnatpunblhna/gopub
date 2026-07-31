package controllers

import (
	"github.com/labstack/echo/v4"
	"github.com/linclin/gopub/src/library/db"
	"github.com/linclin/gopub/src/models"
)

func TestApi(c echo.Context) error {
	ctx := New(c)
	var projects []models.Project
	db.QueryRows(&projects, "SELECT * FROM `project`  WHERE 1=1")
	return ctx.Json(projects)
}
