package models

import (
	"fmt"

	"github.com/linclin/gopub/src/library/db"
)

type ApiSystem struct {
	Id         int    `gorm:"column:AppId;primaryKey"`
	AppSecret  string `gorm:"column:AppSecret;size:255"`
	SystemName string `gorm:"column:SystemName;size:255"`
	IP         string `gorm:"column:IP;size:255"`
	Operator   string `gorm:"column:Operator;size:255"`
}

func (t *ApiSystem) TableName() string {
	return "api_system"
}

// AddApiSystem insert a new ApiSystem into database and returns
// last inserted Id on success.
func AddApiSystem(m *ApiSystem) (id int64, err error) {
	if err = db.DB().Create(m).Error; err != nil {
		return 0, err
	}
	return int64(m.Id), nil
}

// GetApiSystemById retrieves ApiSystem by Id. Returns error if
// Id doesn't exist
func GetApiSystemById(id int) (v *ApiSystem, err error) {
	v = &ApiSystem{}
	if err = db.DB().First(v, id).Error; err != nil {
		return nil, err
	}
	return v, nil
}

// GetAllApiSystem retrieves all ApiSystem matches certain condition. Returns empty list if
// no records exist
func GetAllApiSystem(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	return getAll[ApiSystem](query, fields, sortby, order, offset, limit)
}

// UpdateApiSystem updates ApiSystem by Id and returns error if
// the record to be updated doesn't exist
func UpdateApiSystemById(m *ApiSystem) (err error) {
	// 先确认记录存在，保持原 beego 版本"记录不存在则返回错误"的语义
	if err = db.DB().First(&ApiSystem{}, m.Id).Error; err != nil {
		return err
	}
	tx := db.DB().Save(m)
	if tx.Error == nil {
		fmt.Println("Number of records updated in database:", tx.RowsAffected)
	}
	return tx.Error
}

// DeleteApiSystem deletes ApiSystem by Id and returns error if
// the record to be deleted doesn't exist
func DeleteApiSystem(id int) (err error) {
	if err = db.DB().First(&ApiSystem{}, id).Error; err != nil {
		return err
	}
	tx := db.DB().Delete(&ApiSystem{}, id)
	if tx.Error == nil {
		fmt.Println("Number of records deleted in database:", tx.RowsAffected)
	}
	return tx.Error
}
