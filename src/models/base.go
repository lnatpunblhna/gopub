package models

import (
	"fmt"
	"reflect"

	"github.com/linclin/gopub/src/library/db"
)

// AllModels 汇总全部数据模型，替代 beego 的 orm.RegisterModel 注册机制，
// 供建表（AutoMigrate）时使用。
func AllModels() []interface{} {
	return []interface{}{
		new(User),
		new(Project),
		new(Task),
		new(TaskErrLog),
		new(Record),
		new(Group),
		new(Session),
		new(Migration),
		new(ApiSystem),
	}
}

// getAll 是各 GetAllXxx 的公共实现，保持 beego 版本的入参语义：
// query 为 field__exp 过滤条件，fields 指定返回列，sortby/order 配对排序，
// offset/limit 分页（limit 为负表示不限制）。
func getAll[T any](query map[string]string, fields []string, sortby []string,
	order []string, offset int64, limit int64) (ml []interface{}, err error) {

	var model T
	tx := db.DB().Model(&model)

	if tx, err = db.ApplyFilters(tx, &model, query); err != nil {
		return nil, err
	}

	orderBy, err := db.BuildOrderBy(&model, sortby, order)
	if err != nil {
		return nil, err
	}
	if orderBy != "" {
		tx = tx.Order(orderBy)
	}

	if len(fields) > 0 {
		cols, err := db.Columns(&model)
		if err != nil {
			return nil, err
		}
		for _, f := range fields {
			if _, ok := cols[f]; !ok {
				// 也允许直接传 Go 字段名
				if _, ok := reflect.TypeOf(model).FieldByName(f); !ok {
					return nil, fmt.Errorf("Error: 不支持的返回字段 %q", f)
				}
			}
		}
		tx = tx.Select(fields)
	}

	if limit > 0 {
		tx = tx.Limit(int(limit))
	}
	if offset > 0 {
		tx = tx.Offset(int(offset))
	}

	var list []T
	if err := tx.Find(&list).Error; err != nil {
		return nil, err
	}

	if len(fields) == 0 {
		for _, v := range list {
			ml = append(ml, v)
		}
		return ml, nil
	}

	// 指定了 fields 时只返回这些字段，保持 beego 版本的裁剪行为
	for _, v := range list {
		m := make(map[string]interface{}, len(fields))
		val := reflect.ValueOf(v)
		for _, fname := range fields {
			fv := val.FieldByName(fname)
			if !fv.IsValid() {
				// 传入的是列名时，按列名反查对应的结构体字段
				fv = fieldByColumn(val, fname)
			}
			if fv.IsValid() {
				m[fname] = fv.Interface()
			}
		}
		ml = append(ml, m)
	}
	return ml, nil
}

// fieldByColumn 依据 orm/gorm tag 里的列名反查结构体字段
func fieldByColumn(val reflect.Value, column string) reflect.Value {
	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		if col, ok := columnNameOf(t.Field(i)); ok && col == column {
			return val.Field(i)
		}
	}
	return reflect.Value{}
}

// columnNameOf 从字段的 gorm tag 中解析出列名
func columnNameOf(f reflect.StructField) (string, bool) {
	tag := f.Tag.Get("gorm")
	for _, part := range splitTag(tag) {
		if len(part) > len("column:") && part[:len("column:")] == "column:" {
			return part[len("column:"):], true
		}
	}
	return "", false
}

func splitTag(tag string) []string {
	var out []string
	start := 0
	for i := 0; i < len(tag); i++ {
		if tag[i] == ';' {
			out = append(out, tag[start:i])
			start = i + 1
		}
	}
	return append(out, tag[start:])
}
