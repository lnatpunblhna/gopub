package db

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// 本文件把 beego orm 的查询 DSL（Filter 的 field__exp 语法、OrderBy 的 -field 语法）
// 翻译成 GORM 查询，供各 model 的 GetAllXxx 复用。
//
// 过滤条件的 key 来自外部 API 传入（如 /v1/task?query=status:3），
// beego 的 QueryTable 会校验字段是否存在，这里同样基于模型 schema 做列名白名单，
// 避免把用户输入直接拼进 SQL。

var schemaCache = &sync.Map{}

// Columns 解析模型的合法列名集合
func Columns(model interface{}) (map[string]struct{}, error) {
	return columnsOf(model)
}

// columnsOf 解析模型的合法列名集合
func columnsOf(model interface{}) (map[string]struct{}, error) {
	s, err := schema.Parse(model, schemaCache, schema.NamingStrategy{SingularTable: true})
	if err != nil {
		return nil, err
	}
	cols := make(map[string]struct{}, len(s.FieldsByDBName))
	for name := range s.FieldsByDBName {
		cols[name] = struct{}{}
	}
	return cols, nil
}

// ApplyFilters 把 beego 风格的过滤条件应用到查询上。
// key 形如 `status`、`name__contains`、`id__in`、`memo__isnull`。
func ApplyFilters(tx *gorm.DB, model interface{}, query map[string]string) (*gorm.DB, error) {
	if len(query) == 0 {
		return tx, nil
	}
	cols, err := columnsOf(model)
	if err != nil {
		return nil, err
	}

	for k, v := range query {
		// beego 用 __ 分隔字段与操作符；本项目模型无关联定义，故首段即列名
		col, exp := k, "exact"
		if i := strings.Index(k, "__"); i >= 0 {
			col, exp = k[:i], k[i+2:]
		}
		if _, ok := cols[col]; !ok {
			return nil, fmt.Errorf("Error: 不支持的查询字段 %q", col)
		}
		quoted := "`" + col + "`"

		switch exp {
		case "exact", "iexact":
			tx = tx.Where(quoted+" = ?", v)
		case "contains", "icontains":
			tx = tx.Where(quoted+" LIKE ?", "%"+v+"%")
		case "startswith", "istartswith":
			tx = tx.Where(quoted+" LIKE ?", v+"%")
		case "endswith", "iendswith":
			tx = tx.Where(quoted+" LIKE ?", "%"+v)
		case "gt":
			tx = tx.Where(quoted+" > ?", v)
		case "gte":
			tx = tx.Where(quoted+" >= ?", v)
		case "lt":
			tx = tx.Where(quoted+" < ?", v)
		case "lte":
			tx = tx.Where(quoted+" <= ?", v)
		case "in":
			tx = tx.Where(quoted+" IN (?)", strings.Split(v, "|"))
		case "isnull":
			if v == "true" || v == "1" {
				tx = tx.Where(quoted + " IS NULL")
			} else {
				tx = tx.Where(quoted + " IS NOT NULL")
			}
		default:
			return nil, fmt.Errorf("Error: 不支持的查询操作符 %q", exp)
		}
	}
	return tx, nil
}

// BuildOrderBy 复刻 beego GetAllXxx 中 sortby/order 的配对规则，返回 SQL 的 ORDER BY 子句。
// 规则：两者等长时逐一配对；order 只有一个时作用于所有 sortby 字段；否则报错。
func BuildOrderBy(model interface{}, sortby []string, order []string) (string, error) {
	if len(sortby) == 0 {
		if len(order) != 0 {
			return "", errors.New("Error: unused 'order' fields")
		}
		return "", nil
	}

	cols, err := columnsOf(model)
	if err != nil {
		return "", err
	}

	pick := func(i int) (string, error) {
		switch {
		case len(sortby) == len(order):
			return order[i], nil
		case len(order) == 1:
			return order[0], nil
		default:
			return "", errors.New("Error: 'sortby', 'order' sizes mismatch or 'order' size is not 1")
		}
	}

	var parts []string
	for i, field := range sortby {
		if _, ok := cols[field]; !ok {
			return "", fmt.Errorf("Error: 不支持的排序字段 %q", field)
		}
		o, err := pick(i)
		if err != nil {
			return "", err
		}
		switch o {
		case "desc":
			parts = append(parts, "`"+field+"` DESC")
		case "asc":
			parts = append(parts, "`"+field+"` ASC")
		default:
			return "", errors.New("Error: Invalid order. Must be either [asc|desc]")
		}
	}
	return strings.Join(parts, ", "), nil
}
