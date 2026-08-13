package models

import (
	"fmt"
	"time"

	"github.com/lnatpunblhna/gopub/src/library/db"
)

type TaskErrLog struct {
	Id         int       `gorm:"column:id;primaryKey;autoIncrement"`
	TaskId     int       `gorm:"column:task_id"`
	ErrInfo    string    `gorm:"column:err_info"`
	CreateTime time.Time `gorm:"column:create_time;type:timestamp;autoUpdateTime;serializer:nulltime"`
}

func (t *TaskErrLog) TableName() string {
	return "task_err_log"
}

// AddTaskErrLog insert a new TaskErrLog into database and returns
// last inserted Id on success.
func AddTaskErrLog(m *TaskErrLog) (id int64, err error) {
	if err = db.DB().Create(m).Error; err != nil {
		return 0, err
	}
	return int64(m.Id), nil
}

// GetTaskErrLogById retrieves TaskErrLog by Id. Returns error if
// Id doesn't exist
func GetTaskErrLogById(id int) (v *TaskErrLog, err error) {
	v = &TaskErrLog{}
	if err = db.DB().First(v, id).Error; err != nil {
		return nil, err
	}
	return v, nil
}

// GetAllTaskErrLog retrieves all TaskErrLog matches certain condition. Returns empty list if
// no records exist
func GetAllTaskErrLog(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	return getAll[TaskErrLog](query, fields, sortby, order, offset, limit)
}

// UpdateTaskErrLog updates TaskErrLog by Id and returns error if
// the record to be updated doesn't exist
func UpdateTaskErrLogById(m *TaskErrLog) (err error) {
	// 先确认记录存在，保持原 beego 版本"记录不存在则返回错误"的语义
	if err = db.DB().First(&TaskErrLog{}, m.Id).Error; err != nil {
		return err
	}
	tx := db.DB().Save(m)
	if tx.Error == nil {
		fmt.Println("Number of records updated in database:", tx.RowsAffected)
	}
	return tx.Error
}

// DeleteTaskErrLog deletes TaskErrLog by Id and returns error if
// the record to be deleted doesn't exist
func DeleteTaskErrLog(id int) (err error) {
	if err = db.DB().First(&TaskErrLog{}, id).Error; err != nil {
		return err
	}
	tx := db.DB().Delete(&TaskErrLog{}, id)
	if tx.Error == nil {
		fmt.Println("Number of records deleted in database:", tx.RowsAffected)
	}
	return tx.Error
}
