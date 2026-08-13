package confcontrollers

import (
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/controllers"
	"github.com/lnatpunblhna/gopub/src/library/common"
	"github.com/lnatpunblhna/gopub/src/library/jumpserver"
	"github.com/lnatpunblhna/gopub/src/library/logger"
)

func GroupInfo(c echo.Context) error {
	ctx := controllers.New(c)
	if ctx.User == nil || ctx.User.Id == 0 {
		return ctx.SetJson(2, nil, "not login")
	}
	groupid := ctx.GetString("hostgroup")
	if groupid == "" {
		return ctx.SetJson(1, nil, "params")
	}
	aGroupid := strings.Split(groupid, " ")
	if len(aGroupid) < 1 {
		return ctx.SetJson(1, nil, "params array")
	}

	mGroupid2true := make(map[string]bool)
	var rsIps []string
	for _, gid := range aGroupid {
		aIp, err := jumpserver.GetIpsByGroupid(gid)
		if err != nil {
			logger.Error("获取 jumpserver 节点", gid, "的资产失败:", err)
			return ctx.SetJson(1, nil, "获取分组信息失败："+err.Error())
		}
		logger.Info(aIp)
		mGroupid2true[gid] = true
		for ip := range aIp {
			rsIps = append(rsIps, ip)
		}
	}
	rsIps = common.ArrayUnique(rsIps)

	rsId2Groupname := make(map[string]string)
	group2id, err := jumpserver.GetGroups()
	if err != nil {
		logger.Error("获取 jumpserver 服务器分组失败:", err)
		return ctx.SetJson(1, nil, "获取分组信息失败："+err.Error())
	}
	for group_id, groupname := range group2id {
		if _, ok := mGroupid2true[group_id]; ok {
			rsId2Groupname[group_id] = groupname
		}
	}

	rs := make(map[string]interface{})
	rs["id2groupname"] = rsId2Groupname
	rs["ips"] = rsIps

	return ctx.SetJson(0, rs, "")
}
