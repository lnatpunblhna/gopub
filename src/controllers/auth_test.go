package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lnatpunblhna/gopub/src/models"
)

// newCtxAs 构造一条已经解析出登录用户的请求上下文。
// 直接往 userCtxKey 里塞缓存，绕开 userByToken 的查库，
// 使中间件本身的分支可以脱离数据库单测。user 为 nil 表示未登录。
func newCtxAs(user *models.User) (echo.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	c.Set(userCtxKey, userCache{user: user})
	return c, rec
}

// reached 记录 handler 有没有被真正执行到
func reached(hit *bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		*hit = true
		return c.NoContent(http.StatusOK)
	}
}

// TestRequireAdmin 覆盖三种身份：普通用户必须被挡在管理员接口之外。
// 这是「菜单隐藏了但直接敲 URL 就能进」那类越权的最后一道闸。
func TestRequireAdmin(t *testing.T) {
	cases := []struct {
		name     string
		user     *models.User
		wantPass bool
		wantCode string // 未通过时响应里应出现的业务码
	}{
		{"管理员放行", &models.User{Id: 1, Role: roleAdmin}, true, ""},
		{"预发布用户拒绝", &models.User{Id: 2, Role: rolePreRelease}, false, `"code":1`},
		{"单项目用户拒绝", &models.User{Id: 3, Role: roleSingle}, false, `"code":1`},
		{"未知角色拒绝", &models.User{Id: 4, Role: 0}, false, `"code":1`},
		{"未登录拒绝", nil, false, `"code":2`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, rec := newCtxAs(c.user)
			hit := false
			if err := RequireAdmin(reached(&hit))(ctx); err != nil {
				t.Fatalf("中间件返回错误: %v", err)
			}
			if hit != c.wantPass {
				t.Errorf("handler 执行情况 = %v, want %v", hit, c.wantPass)
			}
			if !c.wantPass && !strings.Contains(rec.Body.String(), c.wantCode) {
				t.Errorf("响应 = %s, 应包含 %s", rec.Body.String(), c.wantCode)
			}
		})
	}
}

// TestRequireLogin 确认登录闸只看登录态，不看角色。
func TestRequireLogin(t *testing.T) {
	for _, c := range []struct {
		name     string
		user     *models.User
		wantPass bool
	}{
		{"普通用户放行", &models.User{Id: 2, Role: rolePreRelease}, true},
		{"未登录拒绝", nil, false},
		{"Id 为 0 视为未登录", &models.User{Id: 0, Role: roleAdmin}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx, _ := newCtxAs(c.user)
			hit := false
			if err := RequireLogin(reached(&hit))(ctx); err != nil {
				t.Fatalf("中间件返回错误: %v", err)
			}
			if hit != c.wantPass {
				t.Errorf("handler 执行情况 = %v, want %v", hit, c.wantPass)
			}
		})
	}
}
