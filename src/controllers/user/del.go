package usercontrollers

import (
	"github.com/astaxie/beego/orm"
	"github.com/linclin/gopub/src/controllers"
	"github.com/linclin/gopub/src/models"
)

type DelController struct {
	controllers.BaseController
}

func (c *DelController) Get() {
	userId, _ := c.GetInt("id", 0)
	if userId == 0 {
		c.SetJson(1, nil, "参数错误")
		return
	}
	err := models.DeleteUser(userId)
	if err != nil {
		c.SetJson(1, nil, err.Error())
		return
	}
	//清理该用户与项目的关联
	o := orm.NewOrm()
	o.Raw("DELETE FROM `group` WHERE `user_id` = ? ", userId).Exec()
	c.SetJson(0, nil, "删除成功")
	return
}
