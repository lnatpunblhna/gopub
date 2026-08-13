package controllers

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
	"github.com/lnatpunblhna/gopub/src/models"
	"golang.org/x/crypto/bcrypt"
)

// 邮箱正则
func IsEmail(str ...string) bool {
	var b bool
	for _, s := range str {
		b, _ = regexp.MatchString("^([a-z0-9_\\.-]+)@([\\da-z\\.-]+)\\.([a-z\\.]{2,6})$", s)
		if false == b {
			return b
		}
	}
	return b
}

func Register(c echo.Context) error {
	ctx := New(c)

	logger.Info(string(ctx.RequestBody()))
	registerData := map[string]interface{}{"user_password": "", "user_name": "", "Role": 1}
	err := json.Unmarshal(ctx.RequestBody(), &registerData)
	if err != nil {
		return ctx.SetJson(1, nil, "数据格式错误")
	}
	registerUsername := common.GetString(registerData["register_username"])
	registerRealname := common.GetString(registerData["register_realname"])
	registerEmail := common.GetString(registerData["register_email"])

	registerRole := common.GetInt(registerData["Role"])

	iseamil := IsEmail(registerEmail)
	if iseamil == false {
		return ctx.SetJson(1, nil, "邮箱输入有误")
	}

	var user models.User
	//先判断存在用户否
	err = db.QueryRow(&user, "SELECT * FROM `user` WHERE username= ?", registerUsername)
	logger.Info(user)
	if err == nil {
		userId, _ := ctx.GetInt("id")
		if userId == 0 {
			return ctx.SetJson(1, nil, "用户已存在，请更换账户名")
		}
		if userId != user.Id {
			return ctx.SetJson(1, nil, "用户不存在")
		}
		user.Role = int16(registerRole)
		user.Email = registerEmail
		user.Realname = registerRealname
		if user.Role == 20 {
			db.Exec("DELETE FROM  `group` WHERE `user_id` =  ? ", user.Id)
			pro_ids := common.GetString(registerData["pro_ids"])
			pro_idArr := strings.Split(pro_ids, ",")
			for _, pro_id := range pro_idArr {
				db.Exec("INSERT INTO `group`(`project_id`, `user_id`) VALUES (?, ?)", pro_id, user.Id)
			}
		}
		if user.FromLdap == 0 {
			if err := models.UpdateUserById(&user); err != nil {
				return ctx.SetJson(1, nil, "数据库存储错误")
			}
			return ctx.SetJson(0, nil, "success")
		}
		return ctx.SetJson(0, nil, "success")
	}

	//不存在，存库
	var newuser models.User
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	newuser.Username = registerUsername
	newuser.PasswordHash = string(hashedPassword)
	newuser.IsEmailVerified = 1
	newuser.Avatar = "default.jpg"
	newuser.Role = int16(registerRole)
	newuser.Status = 10
	newuser.CreatedAt = time.Now()
	newuser.UpdatedAt = time.Now()
	// 不预置 auth_key：新账号还没登录过，预置一个可枚举的值没有用处
	// 且容易被误当成有效凭据。留空，由首次登录时 models.IssueAuthKey 签发。
	newuser.Email = registerEmail
	newuser.Realname = registerRealname

	newid, err := models.AddUser(&newuser)
	if newuser.Role == 20 {
		pro_ids := common.GetString(registerData["pro_ids"])
		pro_idArr := strings.Split(pro_ids, ",")
		for _, pro_id := range pro_idArr {
			db.Exec("INSERT INTO `group`(`project_id`, `user_id`) VALUES (?, ?)", pro_id, newid)
		}
	}
	if err != nil {
		return ctx.SetJson(1, nil, "数据库存储错误")
	}
	return ctx.SetJson(0, nil, "success")
}
