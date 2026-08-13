package apicontrollers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
)

// IssueToken 生成访问 Token（Json Web Token）
//
// 成功: {"access_token":"ACCESS_TOKEN","expires_in":"7200"}
// 失败: {"errcode":"100","errmsg":"invalid appid"}
func IssueToken(c echo.Context) error {
	appid := c.QueryParam("appid")
	appsecret := c.QueryParam("appsecret")
	if appid == "" || appsecret == "" {
		return c.JSON(http.StatusOK, map[string]string{"errcode": "100", "errmsg": "appid & appsecret必须 "})
	}
	clientip := c.RealIP()
	api_system, err := models.GetApiSystemById(common.StrToInt(appid))
	if err != nil {
		return c.JSON(http.StatusOK, map[string]string{"errcode": "100", "errmsg": "appid不存在 "})
	}
	if api_system.AppSecret != appsecret {
		return c.JSON(http.StatusOK, map[string]string{"errcode": "100", "errmsg": "appid 或者 appsecret不匹配  "})
	}
	api_system_ips := strings.Split(api_system.IP, ",")
	logger.Debug(api_system_ips)
	ipin := false
	for _, ip := range api_system_ips {
		if clientip == ip {
			ipin = true
		}
	}
	if !ipin {
		return c.JSON(http.StatusOK, map[string]string{"errcode": "100", "errmsg": clientip + " 请求IP不在允许范围内  "})
	}
	// Create a Token that will be signed with HS256.
	expat := time.Now().Add(time.Hour)
	exptime := expat.Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": appid,                     //The issuer of the token，token 是给谁的
		"exp": jwt.NewNumericDate(expat), //Expiration Time。 token 过期时间，Unix 时间戳格式
	})
	// The claims object allows you to store information in the actual token.
	tokenString, err := token.SignedString([]byte(config.String("SecretKey")))
	// tokenString Contains the actual token you should share with your client.
	if err != nil {
		return c.JSON(http.StatusOK, map[string]string{"errcode": "100", "errmsg": "Token生成错误:" + err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"access_token": tokenString, "expires_in": strconv.FormatInt(exptime, 10)})
}
