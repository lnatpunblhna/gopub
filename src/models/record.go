package models

import (
	"fmt"

	"github.com/linclin/gopub/src/library/db"
)

type Record struct {
	Id        int    `gorm:"column:id;primaryKey;autoIncrement"`
	UserId    uint   `gorm:"column:user_id"`
	TaskId    int64  `gorm:"column:task_id"`
	Status    int16  `gorm:"column:status"`
	Action    uint   `gorm:"column:action"`
	Command   string `gorm:"column:command;type:text"`
	Duration  int    `gorm:"column:duration"`
	Memo      string `gorm:"column:memo;type:text"`
	CreatedAt int    `gorm:"column:created_at;autoCreateTime:false"`
}

func (t *Record) TableName() string {
	return "record"
}

// AddRecord insert a new Record into database and returns
// last inserted Id on success.
func AddRecord(m *Record) (id int64, err error) {
	if err = db.DB().Create(m).Error; err != nil {
		return 0, err
	}
	return int64(m.Id), nil
}

// GetRecordById retrieves Record by Id. Returns error if
// Id doesn't exist
func GetRecordById(id int) (v *Record, err error) {
	v = &Record{}
	if err = db.DB().First(v, id).Error; err != nil {
		return nil, err
	}
	return v, nil
}

// GetAllRecord retrieves all Record matches certain condition. Returns empty list if
// no records exist
func GetAllRecord(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	return getAll[Record](query, fields, sortby, order, offset, limit)
}

// UpdateRecord updates Record by Id and returns error if
// the record to be updated doesn't exist
func UpdateRecordById(m *Record) (err error) {
	// 先确认记录存在，保持原 beego 版本"记录不存在则返回错误"的语义
	if err = db.DB().First(&Record{}, m.Id).Error; err != nil {
		return err
	}
	tx := db.DB().Save(m)
	if tx.Error == nil {
		fmt.Println("Number of records updated in database:", tx.RowsAffected)
	}
	return tx.Error
}

// DeleteRecord deletes Record by Id and returns error if
// the record to be deleted doesn't exist
func DeleteRecord(id int) (err error) {
	if err = db.DB().First(&Record{}, id).Error; err != nil {
		return err
	}
	tx := db.DB().Delete(&Record{}, id)
	if tx.Error == nil {
		fmt.Println("Number of records deleted in database:", tx.RowsAffected)
	}
	return tx.Error
}
