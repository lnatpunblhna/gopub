package models

import (
	"fmt"
	"time"

	"github.com/lnatpunblhna/gopub/src/library/db"
)

type User struct {
	Id              int    `gorm:"column:id;primaryKey;autoIncrement"`
	Username        string `gorm:"column:username;size:255"`
	IsEmailVerified int8   `gorm:"column:is_email_verified"`
	AuthKey         string `gorm:"column:auth_key;size:32"`
	// AuthKeyExpireAt 为 auth_key 的过期时刻，NULL 表示没有有效凭据。
	// 用指针是为了让"无凭据"写成 NULL 而不是 '0000-00-00'（严格模式下会报错）。
	AuthKeyExpireAt        *time.Time `gorm:"column:auth_key_expire_at;type:datetime"`
	PasswordHash           string     `gorm:"column:password_hash;size:255"`
	PasswordResetToken     string     `gorm:"column:password_reset_token;size:255"`
	EmailConfirmationToken string     `gorm:"column:email_confirmation_token;size:255"`
	Email                  string     `gorm:"column:email;size:255"`
	Avatar                 string     `gorm:"column:avatar;size:100"`
	Role                   int16      `gorm:"column:role"`
	FromLdap               int16      `gorm:"column:from_ldap"`
	Status                 int16      `gorm:"column:status"`
	CreatedAt              time.Time  `gorm:"column:created_at;type:datetime;autoCreateTime:false"`
	UpdatedAt              time.Time  `gorm:"column:updated_at;type:datetime;autoUpdateTime:false"`
	Realname               string     `gorm:"column:realname;size:32"`
}

func (t *User) TableName() string {
	return "user"
}

// AddUser insert a new User into database and returns
// last inserted Id on success.
func AddUser(m *User) (id int64, err error) {
	if err = db.DB().Create(m).Error; err != nil {
		return 0, err
	}
	return int64(m.Id), nil
}

// GetUserById retrieves User by Id. Returns error if
// Id doesn't exist
func GetUserById(id int) (v *User, err error) {
	v = &User{}
	if err = db.DB().First(v, id).Error; err != nil {
		return nil, err
	}
	return v, nil
}

// GetAllUser retrieves all User matches certain condition. Returns empty list if
// no records exist
func GetAllUser(query map[string]string, fields []string, sortby []string, order []string,
	offset int64, limit int64) (ml []interface{}, err error) {
	return getAll[User](query, fields, sortby, order, offset, limit)
}

// UpdateUser updates User by Id and returns error if
// the record to be updated doesn't exist
func UpdateUserById(m *User) (err error) {
	// 先确认记录存在，保持原 beego 版本"记录不存在则返回错误"的语义
	if err = db.DB().First(&User{}, m.Id).Error; err != nil {
		return err
	}
	tx := db.DB().Save(m)
	if tx.Error == nil {
		fmt.Println("Number of records updated in database:", tx.RowsAffected)
	}
	return tx.Error
}

// DeleteUser deletes User by Id and returns error if
// the record to be deleted doesn't exist
func DeleteUser(id int) (err error) {
	if err = db.DB().First(&User{}, id).Error; err != nil {
		return err
	}
	tx := db.DB().Delete(&User{}, id)
	if tx.Error == nil {
		fmt.Println("Number of records deleted in database:", tx.RowsAffected)
	}
	return tx.Error
}
