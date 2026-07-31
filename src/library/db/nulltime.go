package db

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"gorm.io/gorm/schema"
)

// nullTimeSerializer 让零值 time.Time 以 NULL 写入数据库。
//
// 原 beego 模型把 project/task 的时间列标记为 null，beego 写库时会把零值
// time.Time 转成 NULL；GORM 默认原样写入 0001-01-01，会被 MySQL 的 DATETIME
// 取值范围拒绝（Error 1292）。这里用 serializer 复刻 beego 的行为，
// 字段类型仍是 time.Time，因此接口返回的 JSON 结构不受影响。
//
// 在模型上通过 `gorm:"...;serializer:nulltime"` 启用。
type nullTimeSerializer struct{}

func init() {
	schema.RegisterSerializer("nulltime", nullTimeSerializer{})
}

// Value 写库前转换：零值时间写 NULL
func (nullTimeSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	switch v := fieldValue.(type) {
	case time.Time:
		if v.IsZero() {
			return nil, nil
		}
		return v, nil
	case *time.Time:
		if v == nil || v.IsZero() {
			return nil, nil
		}
		return *v, nil
	case nil:
		return nil, nil
	}
	return fieldValue, nil
}

// Scan 读库后转换：NULL 读成零值时间，与 beego 的表现一致
func (nullTimeSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	var t time.Time

	switch v := dbValue.(type) {
	case nil:
		// 保持零值
	case time.Time:
		t = v
	case *time.Time:
		if v != nil {
			t = *v
		}
	case []byte:
		parsed, err := parseDBTime(string(v))
		if err != nil {
			return err
		}
		t = parsed
	case string:
		parsed, err := parseDBTime(v)
		if err != nil {
			return err
		}
		t = parsed
	default:
		return fmt.Errorf("无法把 %T 解析为时间字段 %s", dbValue, field.Name)
	}

	field.ReflectValueOf(ctx, dst).Set(reflect.ValueOf(t))
	return nil
}

// parseDBTime 按 MySQL 常见的时间文本格式解析
func parseDBTime(s string) (time.Time, error) {
	if s == "" || s == "0000-00-00 00:00:00" {
		return time.Time{}, nil
	}
	for _, layout := range []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("无法解析时间: %q", s)
}
