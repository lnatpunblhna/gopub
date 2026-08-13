package apicontrollers

import (
	"net/http"
	"runtime"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/logger"
)

// appIdKey 是 JWT 签发方（appid）在请求上下文中的键。
// 原实现用包级变量 var AppId int 保存，多请求并发时会互相覆盖，改为按请求存取。
const appIdKey = "gopub_app_id"

// AppIdFrom 返回当前请求经 JWT 校验得到的 appid
func AppIdFrom(c echo.Context) int {
	if v, ok := c.Get(appIdKey).(int); ok {
		return v
	}
	return 0
}

// ApiAuth 校验 Json Web Token，替代原 BaseApiController.Prepare。
func ApiAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if panicErr := recover(); panicErr != nil {
				buf := make([]byte, 1024)
				n := runtime.Stack(buf, false)
				logger.Error("控制器错误:", panicErr, string(buf[0:n]))
			}
		}()

		//正式环境则需要验证Token并且需要使用HTTPS访问
		//if config.RunMode() == "prod" {
		//	if !c.IsTLS() {
		//		return c.JSON(http.StatusOK, map[string]string{"errcode": "101", "errmsg": "请使用HTTPS请求API"})
		//	}
		//}

		//验证token
		tokenString := c.Request().Header.Get("Authorization")
		if tokenString == "" {
			return c.JSON(http.StatusOK, map[string]string{"errcode": "102", "errmsg": "token错误"})
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.String("SecretKey")), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		// 原实现在 claims 断言失败时会放行请求，这里统一按验证失败处理
		if err != nil {
			return c.JSON(http.StatusOK, map[string]string{"errcode": "103", "errmsg": "token验证失败"})
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.JSON(http.StatusOK, map[string]string{"errcode": "103", "errmsg": "token验证失败"})
		}
		c.Set(appIdKey, common.GetInt(claims["iss"]))
		return next(c)
	}
}
