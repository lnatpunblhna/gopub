package models

// 本文件集中管理控制台登录凭据 auth_key 的生命周期。
//
// auth_key 由前端存进 cookie，并以 `Authorization: TOKEN xxx` 随请求携带，
// 服务端在 src/controllers/base.go 的 userByToken 里解析。这里负责三件事：
//   - 签发：用 crypto/rand 生成，取代原先 md5(用户名 + 秒级时间戳) 的可枚举取值
//   - 续期：滑动过期，有请求就顺延，闲置超过 authKeyLifetime 后失效
//   - 吊销：登出与改密码时清空，使已发出的凭据立即失效
//
// 同一账号多端登录不会互踢：登录时若已有未过期的 auth_key 就沿用原值、只顺延
// 过期时间；登出清空 auth_key，则所有端一起失效。

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/lnatpunblhna/gopub/src/library/config"
	"github.com/lnatpunblhna/gopub/src/library/db"
	"github.com/lnatpunblhna/gopub/src/library/logger"
)

// authKeyColumn 是过期时刻对应的列名，建列与更新都用它
const authKeyColumn = "auth_key_expire_at"

// defaultAuthKeyLifetime 是 app.conf 未配置 authKeyLifetime 时的默认有效期
const defaultAuthKeyLifetime = 7 * 24 * time.Hour

// AuthKeyLifetime 返回登录凭据有效期，取 app.conf 的 authKeyLifetime（单位秒）。
// 配置值非正数时回落到默认值，避免误配成 0 导致登录后立刻失效。
func AuthKeyLifetime() time.Duration {
	if sec := config.DefaultInt("authKeyLifetime", 0); sec > 0 {
		return time.Duration(sec) * time.Second
	}
	return defaultAuthKeyLifetime
}

// newAuthKey 生成 32 位十六进制随机串，长度对齐 user.auth_key 的 size:32
func newAuthKey() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// AuthKeyValid 判断凭据是否仍在有效期内。
// 过期时刻为 NULL 说明这是升级前遗留的永久 token（或库里还没建这一列），
// 一律视为无效，强制重新登录。
func AuthKeyValid(u *User) bool {
	if u == nil || u.AuthKey == "" || u.AuthKeyExpireAt == nil {
		return false
	}
	return time.Now().Before(*u.AuthKeyExpireAt)
}

// IssueAuthKey 在登录成功后签发凭据并写库，同时把新值回填到 u。
// 已有凭据仍有效时沿用原 auth_key，只顺延过期时间，从而不踢掉其他已登录的端。
func IssueAuthKey(u *User) error {
	if u == nil || u.Id == 0 {
		return nil
	}
	if !AuthKeyValid(u) {
		key, err := newAuthKey()
		if err != nil {
			return err
		}
		u.AuthKey = key
	}
	expireAt := time.Now().Add(AuthKeyLifetime())
	if err := saveAuthKey(u.Id, u.AuthKey, expireAt); err != nil {
		return err
	}
	u.AuthKeyExpireAt = &expireAt
	return nil
}

// RefreshAuthKey 实现滑动过期：仅当剩余有效期不足一半时才写库顺延，
// 避免每个请求都产生一次 UPDATE。续期失败不影响本次请求。
func RefreshAuthKey(u *User) {
	if u == nil || u.Id == 0 || u.AuthKeyExpireAt == nil {
		return
	}
	lifetime := AuthKeyLifetime()
	if time.Until(*u.AuthKeyExpireAt) > lifetime/2 {
		return
	}
	expireAt := time.Now().Add(lifetime)
	if err := saveAuthKey(u.Id, u.AuthKey, expireAt); err != nil {
		logger.Error("登录凭据续期失败:", err)
		return
	}
	u.AuthKeyExpireAt = &expireAt
}

// RevokeAuthKey 清空指定用户的凭据，用于登出与改密码，使旧 token 立即失效
func RevokeAuthKey(id int) error {
	if id == 0 {
		return nil
	}
	return db.DB().Model(&User{}).Where("id = ?", id).
		Updates(map[string]interface{}{"auth_key": "", authKeyColumn: nil}).Error
}

// saveAuthKey 只更新凭据相关的两列。不用 UpdateUserById，因为它走 Save 会整行覆盖，
// 容易把调用方内存里的其他字段（如被清空的 password_hash）一起写回库。
func saveAuthKey(id int, authKey string, expireAt time.Time) error {
	return db.DB().Model(&User{}).Where("id = ?", id).
		Updates(map[string]interface{}{"auth_key": authKey, authKeyColumn: expireAt}).Error
}

// MigrateAuthKeyColumn 补齐 user.auth_key_expire_at 列。
//
// Syncdb 只在 -syncdb / -docker 启动参数下执行，老部署直接替换二进制不会建这一列，
// 而缺列会让登录时的 UPDATE 直接失败，所以每次启动检查一次。
// 没有 DDL 权限之类的失败只记日志，提示手动执行 ./control syncdb。
func MigrateAuthKeyColumn() {
	m := db.DB().Migrator()
	if m.HasColumn(&User{}, authKeyColumn) {
		return
	}
	if err := m.AddColumn(&User{}, authKeyColumn); err != nil {
		logger.Error("user."+authKeyColumn+" 列创建失败，登录会不可用，请手动执行 ./control syncdb:", err)
		return
	}
	logger.Info("已补齐 user." + authKeyColumn + " 列")
}
