package models

import (
	"fmt"
	"time"

	"github.com/linclin/gopub/src/library/db"
)

const RELEASE_TYPE_SOFTLINK = 0
const RELEASE_TYPE_MOVEDIR = 1

type Project struct {
	Id                  int       `gorm:"column:id;primaryKey;autoIncrement"`
	UserId              uint      `gorm:"column:user_id"`
	Name                string    `gorm:"column:name;size:100"`
	Tag                 string    `gorm:"column:tag;size:100"` //标签 用户分组显示
	Level               int16     `gorm:"column:level"`
	Status              int16     `gorm:"column:status"`
	Version             string    `gorm:"column:version;size:32"`
	RepoUrl             string    `gorm:"column:repo_url;type:text"`
	RepoUsername        string    `gorm:"column:repo_username;size:50"`
	RepoPassword        string    `gorm:"column:repo_password;size:100"`
	RepoMode            string    `gorm:"column:repo_mode;size:50"`
	RepoType            string    `gorm:"column:repo_type;size:10"`
	DeployFrom          string    `gorm:"column:deploy_from;size:200"`
	Excludes            string    `gorm:"column:excludes;type:text"`
	ReleaseUser         string    `gorm:"column:release_user;size:50"`
	ReleaseTo           string    `gorm:"column:release_to;size:200"`
	ReleaseLibrary      string    `gorm:"column:release_library;type:text"`
	ReleaseType         int16     `gorm:"column:release_type"` //发布方式 0短链接 1移动目录
	Hosts               string    `gorm:"column:hosts;type:text"`
	PreDeploy           string    `gorm:"column:pre_deploy;type:text"`
	PostDeploy          string    `gorm:"column:post_deploy;type:text"`
	PreRelease          string    `gorm:"column:pre_release;type:text"`
	PostRelease         string    `gorm:"column:post_release;type:text"`
	PostReleaseTogether string    `gorm:"column:post_release_together;type:text"` //所有服务器部属完成后，再统一执行的命令，主要防止单机部属速度不一而导致如服务重启不同时的问题
	LastDeploy          string    `gorm:"column:last_deploy;type:text"`
	Audit               int16     `gorm:"column:audit"`
	KeepVersionNum      int       `gorm:"column:keep_version_num"`
	CreatedAt           time.Time `gorm:"column:created_at;type:datetime;autoCreateTime:false;serializer:nulltime"`
	UpdatedAt           time.Time `gorm:"column:updated_at;type:datetime;autoUpdateTime:false;serializer:nulltime"`
	ShowHistory         int16     `gorm:"column:view_history"` //显示较前次上线的代码变更
	P2p                 int16     `gorm:"column:p2p"`
	SshAlgorithm        int16     `gorm:"column:ssh_algorithm"`
	HostGroup           string    `gorm:"column:host_group"` //服务器分组，基于jumpserver groupid,groupid
	Gzip                int16     `gorm:"column:gzip"`
	IsGroup             int16     `gorm:"column:is_group"`
	UserLock            int       `gorm:"column:user_lock"` //用户锁定 uid
	PmsProName          string    `gorm:"column:pms_pro_name;size:200"`
}

func (t *Project) TableName() string {
	return "project"
}

// AddProject insert a new Project into database and returns
// last inserted Id on success.
func AddProject(m *Project) (id int64, err error) {
	if err = db.DB().Create(m).Error; err != nil {
		return 0, err
	}
	return int64(m.Id), nil
}

// GetProjectById retrieves Project by Id. Returns error if
// Id doesn't exist
func GetProjectById(id int) (v *Project, err error) {
	v = &Project{}
	if err = db.DB().First(v, id).Error; err != nil {
		return nil, err
	}
	return v, nil
}

// GetAllProject retrieves all Project matches certain condition. Returns empty list if
// no records exist
func GetAllProject(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	return getAll[Project](query, fields, sortby, order, offset, limit)
}

// UpdateProject updates Project by Id and returns error if
// the record to be updated doesn't exist
func UpdateProjectById(m *Project) (err error) {
	// 先确认记录存在，保持原 beego 版本"记录不存在则返回错误"的语义
	if err = db.DB().First(&Project{}, m.Id).Error; err != nil {
		return err
	}
	tx := db.DB().Save(m)
	if tx.Error == nil {
		fmt.Println("Number of records updated in database:", tx.RowsAffected)
	}
	return tx.Error
}

// DeleteProject deletes Project by Id and returns error if
// the record to be deleted doesn't exist
func DeleteProject(id int) (err error) {
	if err = db.DB().First(&Project{}, id).Error; err != nil {
		return err
	}
	tx := db.DB().Delete(&Project{}, id)
	if tx.Error == nil {
		fmt.Println("Number of records deleted in database:", tx.RowsAffected)
	}
	return tx.Error
}
