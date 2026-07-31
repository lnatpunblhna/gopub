package models

import (
	"fmt"

	"github.com/linclin/gopub/src/library/db"
)

type Migration struct {
	Id        int `gorm:"column:version;primaryKey"`
	ApplyTime int `gorm:"column:apply_time"`
}

func (t *Migration) TableName() string {
	return "migration"
}

// AddMigration insert a new Migration into database and returns
// last inserted Id on success.
func AddMigration(m *Migration) (id int64, err error) {
	if err = db.DB().Create(m).Error; err != nil {
		return 0, err
	}
	return int64(m.Id), nil
}

// GetMigrationById retrieves Migration by Id. Returns error if
// Id doesn't exist
func GetMigrationById(id int) (v *Migration, err error) {
	v = &Migration{}
	if err = db.DB().First(v, id).Error; err != nil {
		return nil, err
	}
	return v, nil
}

// GetAllMigration retrieves all Migration matches certain condition. Returns empty list if
// no records exist
func GetAllMigration(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	return getAll[Migration](query, fields, sortby, order, offset, limit)
}

// UpdateMigration updates Migration by Id and returns error if
// the record to be updated doesn't exist
func UpdateMigrationById(m *Migration) (err error) {
	// 先确认记录存在，保持原 beego 版本"记录不存在则返回错误"的语义
	if err = db.DB().First(&Migration{}, m.Id).Error; err != nil {
		return err
	}
	tx := db.DB().Save(m)
	if tx.Error == nil {
		fmt.Println("Number of records updated in database:", tx.RowsAffected)
	}
	return tx.Error
}

// DeleteMigration deletes Migration by Id and returns error if
// the record to be deleted doesn't exist
func DeleteMigration(id int) (err error) {
	if err = db.DB().First(&Migration{}, id).Error; err != nil {
		return err
	}
	tx := db.DB().Delete(&Migration{}, id)
	if tx.Error == nil {
		fmt.Println("Number of records deleted in database:", tx.RowsAffected)
	}
	return tx.Error
}
