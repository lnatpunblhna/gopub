package models

import (
	"fmt"

	"github.com/linclin/gopub/src/library/db"
)

type Session struct {
	Id     int    `gorm:"column:id;primaryKey"`
	Expire uint   `gorm:"column:expire"`
	Data   string `gorm:"column:data"`
}

func (t *Session) TableName() string {
	return "session"
}

// AddSession insert a new Session into database and returns
// last inserted Id on success.
func AddSession(m *Session) (id int64, err error) {
	if err = db.DB().Create(m).Error; err != nil {
		return 0, err
	}
	return int64(m.Id), nil
}

// GetSessionById retrieves Session by Id. Returns error if
// Id doesn't exist
func GetSessionById(id int) (v *Session, err error) {
	v = &Session{}
	if err = db.DB().First(v, id).Error; err != nil {
		return nil, err
	}
	return v, nil
}

// GetAllSession retrieves all Session matches certain condition. Returns empty list if
// no records exist
func GetAllSession(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	return getAll[Session](query, fields, sortby, order, offset, limit)
}

// UpdateSession updates Session by Id and returns error if
// the record to be updated doesn't exist
func UpdateSessionById(m *Session) (err error) {
	// 先确认记录存在，保持原 beego 版本"记录不存在则返回错误"的语义
	if err = db.DB().First(&Session{}, m.Id).Error; err != nil {
		return err
	}
	tx := db.DB().Save(m)
	if tx.Error == nil {
		fmt.Println("Number of records updated in database:", tx.RowsAffected)
	}
	return tx.Error
}

// DeleteSession deletes Session by Id and returns error if
// the record to be deleted doesn't exist
func DeleteSession(id int) (err error) {
	if err = db.DB().First(&Session{}, id).Error; err != nil {
		return err
	}
	tx := db.DB().Delete(&Session{}, id)
	if tx.Error == nil {
		fmt.Println("Number of records deleted in database:", tx.RowsAffected)
	}
	return tx.Error
}
