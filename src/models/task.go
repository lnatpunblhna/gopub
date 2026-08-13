package models

import (
	"fmt"
	"time"

	"github.com/lnatpunblhna/gopub/src/library/db"
)

type Task struct {
	Id             int       `gorm:"column:id;primaryKey;autoIncrement"`
	UserId         uint      `gorm:"column:user_id"`
	ProjectId      int       `gorm:"column:project_id"`
	Action         int16     `gorm:"column:action"`
	Status         int16     `gorm:"column:status"`
	Title          string    `gorm:"column:title;size:100"`
	LinkId         string    `gorm:"column:link_id;size:20"`
	ExLinkId       string    `gorm:"column:ex_link_id;size:20"`
	CommitId       string    `gorm:"column:commit_id;size:800"`
	CreatedAt      time.Time `gorm:"column:created_at;type:datetime;autoCreateTime:false;serializer:nulltime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:datetime;autoUpdateTime:false;serializer:nulltime"`
	Branch         string    `gorm:"column:branch;size:100"`
	FileList       string    `gorm:"column:file_list"`
	EnableRollback int       `gorm:"column:enable_rollback"`
	PmsBatchId     int       `gorm:"column:pms_batch_id"`
	PmsUworkId     int       `gorm:"column:pms_uwork_id"`
	IsRun          int       `gorm:"column:is_run"`
	FileMd5        string    `gorm:"column:file_md5;size:200"`
	Hosts          string    `gorm:"column:hosts"`
	HostGroup      string    `gorm:"column:host_group"`
}

func (t *Task) TableName() string {
	return "task"
}

// AddTask insert a new Task into database and returns
// last inserted Id on success.
func AddTask(m *Task) (id int64, err error) {
	if err = db.DB().Create(m).Error; err != nil {
		return 0, err
	}
	return int64(m.Id), nil
}

// GetTaskById retrieves Task by Id. Returns error if
// Id doesn't exist
func GetTaskById(id int) (v *Task, err error) {
	v = &Task{}
	if err = db.DB().First(v, id).Error; err != nil {
		return nil, err
	}
	return v, nil
}

// GetAllTask retrieves all Task matches certain condition. Returns empty list if
// no records exist
func GetAllTask(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	return getAll[Task](query, fields, sortby, order, offset, limit)
}

// UpdateTask updates Task by Id and returns error if
// the record to be updated doesn't exist
func UpdateTaskById(m *Task) (err error) {
	// 先确认记录存在，保持原 beego 版本"记录不存在则返回错误"的语义
	if err = db.DB().First(&Task{}, m.Id).Error; err != nil {
		return err
	}
	tx := db.DB().Save(m)
	if tx.Error == nil {
		fmt.Println("Number of records updated in database:", tx.RowsAffected)
	}
	return tx.Error
}

// DeleteTask deletes Task by Id and returns error if
// the record to be deleted doesn't exist
func DeleteTask(id int) (err error) {
	if err = db.DB().First(&Task{}, id).Error; err != nil {
		return err
	}
	tx := db.DB().Delete(&Task{}, id)
	if tx.Error == nil {
		fmt.Println("Number of records deleted in database:", tx.RowsAffected)
	}
	return tx.Error
}

// GetAllTaskAndPro 查询已上线任务及其所属项目名。
// 原实现把入参直接拼进 SQL，这里改为参数化查询避免注入。
func GetAllTaskAndPro(pro_name string, startTime string, endTime string) ([]db.Params, error) {
	sql := "SELECT title,task.id,commit_id,branch,project_id,project.`name` as project_name,task.updated_at FROM task LEFT JOIN project on task.project_id=project.id WHERE action=0 AND task.`status`=3 AND project.level=3 "
	args := []interface{}{}
	if pro_name != "" {
		sql += "AND project.repo_url like ? "
		args = append(args, "%"+pro_name+"%")
	}
	if startTime != "" {
		sql += "and task.updated_at> ? "
		args = append(args, startTime)
	}
	if endTime != "" {
		sql += "and task.updated_at< ? "
		args = append(args, endTime)
	}
	sql += "order by task.id DESC "
	return db.Values(sql, args...)
}
