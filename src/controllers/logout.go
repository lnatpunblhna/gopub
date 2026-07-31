package controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func Logout(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"code": 0, "msg": "sucess"})
}
