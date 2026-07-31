package models

import (
	"fmt"

	"github.com/linclin/gopub/src/library/db"
)

type Group struct {
	Id        int   `gorm:"column:id;primaryKey;autoIncrement"`
	ProjectId uint  `gorm:"column:project_id"`
	UserId    int   `gorm:"column:user_id"`
	Type      int16 `gorm:"column:type"`
}

func (t *Group) TableName() string {
	return "group"
}

// AddGroup insert a new Group into database and returns
// last inserted Id on success.
func AddGroup(m *Group) (id int64, err error) {
	if err = db.DB().Create(m).Error; err != nil {
		return 0, err
	}
	return int64(m.Id), nil
}

// GetGroupById retrieves Group by Id. Returns error if
// Id doesn't exist
func GetGroupById(id int) (v *Group, err error) {
	v = &Group{}
	if err = db.DB().First(v, id).Error; err != nil {
		return nil, err
	}
	return v, nil
}

// GetAllGroup retrieves all Group matches certain condition. Returns empty list if
// no records exist
func GetAllGroup(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	return getAll[Group](query, fields, sortby, order, offset, limit)
}

// UpdateGroup updates Group by Id and returns error if
// the record to be updated doesn't exist
func UpdateGroupById(m *Group) (err error) {
	// 先确认记录存在，保持原 beego 版本"记录不存在则返回错误"的语义
	if err = db.DB().First(&Group{}, m.Id).Error; err != nil {
		return err
	}
	tx := db.DB().Save(m)
	if tx.Error == nil {
		fmt.Println("Number of records updated in database:", tx.RowsAffected)
	}
	return tx.Error
}

// DeleteGroup deletes Group by Id and returns error if
// the record to be deleted doesn't exist
func DeleteGroup(id int) (err error) {
	if err = db.DB().First(&Group{}, id).Error; err != nil {
		return err
	}
	tx := db.DB().Delete(&Group{}, id)
	if tx.Error == nil {
		fmt.Println("Number of records deleted in database:", tx.RowsAffected)
	}
	return tx.Error
}
