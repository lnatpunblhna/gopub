package controllers

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/ldap"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
	"golang.org/x/crypto/bcrypt"
)

func Login(c echo.Context) error {
	ctx := New(c)
	//哈希校验成功后 更新 auth_key
	logger.Info(string(ctx.RequestBody()))
	postData := map[string]string{"user_password": "", "user_name": ""}
	err := json.Unmarshal(ctx.RequestBody(), &postData)
	if err != nil {
		return ctx.SetJson(1, nil, "数据格式错误")
	}
	password := postData["user_password"]
	userName := postData["user_name"]
	if userName == "" || password == "" {
		return ctx.SetJson(1, nil, "用户名或密码不存在")
	}
	var user models.User
	db.QueryRow(&user, "SELECT * FROM `user` WHERE username= ?", userName)
	logger.Info(user)

	enableLdap, _ := config.Bool("enableLdap")
	if enableLdap {
		msg, user, isOk := ldapLogin(userName, password, user)
		if !isOk {
			return ctx.SetJson(1, nil, msg)
		}
		if err := models.IssueAuthKey(&user); err != nil {
			logger.Error("签发登录凭据失败:", err)
			return ctx.SetJson(1, nil, "登录失败，请重试")
		}
		return ctx.SetJson(0, user, "")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return ctx.SetJson(1, nil, "用户名或密码错误")
	}
	if err := models.IssueAuthKey(&user); err != nil {
		logger.Error("签发登录凭据失败:", err)
		return ctx.SetJson(1, nil, "登录失败，请重试")
	}
	user.PasswordHash = ""
	return ctx.SetJson(0, user, "")
}

func ldapLogin(userName string, password string, gopub_user models.User) (msg string, user models.User, isOk bool) {
	ldapClient := ldap.Ldap{}
	e := ldapClient.Connect()
	if e != nil {
		return "ldap连接失败", gopub_user, false
	}
	//验证用户身份
	ldap_user, e := ldapClient.AuthByUidAndPassword(userName, password)
	if e != nil {
		return "ldap身份认证失败", gopub_user, false
	}
	//验证是否在gopub用户组
	ldapGroupFilter := config.String("ldapGroupFilter")
	ldapGroupFilter = strings.Replace(ldapGroupFilter, "{UidNumber}", ldap_user.UidNumber, -1)
	ldapGroupFilter = strings.Replace(ldapGroupFilter, "{uid}", ldap_user.Uid, -1)
	ldapGroupFilter = strings.Replace(ldapGroupFilter, "{cn}", ldap_user.Cn, -1)
	ldapGroupFilter = strings.Replace(ldapGroupFilter, "{sn}", ldap_user.Sn, -1)

	groupCn, e := ldapClient.SearchGroupCn(ldapGroupFilter)
	if e != nil {
		logger.Info("ldap组身份验证失败")
		return "ldap组身份验证失败", gopub_user, false
	}

	role_id64, _ := strconv.ParseInt(config.String("ldapGroupName2roleid_"+groupCn), 10, 64)
	role_id := int16(role_id64)
	// 用户不存在，自动同步进gopub数据库
	if gopub_user.Username == "" {
		addUserFromLdap2Gopub(ldap_user, role_id)
		db.QueryRow(&gopub_user, "SELECT * FROM `user` WHERE username= ?", userName)
		logger.Info(gopub_user)
	} else if role_id != gopub_user.Role {
		//role变更
		gopub_user.Role = role_id
		models.UpdateUserById(&gopub_user)
	}

	return "", gopub_user, true
}

func addUserFromLdap2Gopub(user ldap.Ldap_user, role_id int16) {
	uidNumber, _ := strconv.Atoi(user.UidNumber)

	userModel := models.User{}
	userModel.Id = uidNumber
	userModel.Username = user.Uid
	userModel.Email = user.Email
	userModel.Realname = user.Cn
	userModel.CreatedAt = time.Now()
	userModel.UpdatedAt = time.Now()
	userModel.Avatar = "default.jpg"
	userModel.Role = role_id
	userModel.FromLdap = 1
	uid, _ := models.AddUser(&userModel)
	logger.Info(uid)
}
