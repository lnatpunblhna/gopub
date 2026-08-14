package db

import (
	"testing"
	"time"

	"github.com/lnatpunblhna/gopub/src/library/config"
)

// TestPoolConfigFromConfig 防止连接池配置键名再次与 conf/app.conf 漂移。
// 曾经代码读 mysql_max_idle_conn 而配置文件写 db_max_idle_conn，
// 键名对不上导致池参数一个都没设，静默退回 database/sql 的默认值
// （空闲连接只有 2 条，最大连接数无限制）。
func TestPoolConfigFromConfig(t *testing.T) {
	if err := config.Load("../../conf/app.conf"); err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	for _, mode := range []string{"dev", "prod", "docker"} {
		config.SetRunMode(mode)
		got := PoolConfigFromConfig()

		if got.MaxIdleConn != 30 {
			t.Errorf("[%s] MaxIdleConn = %d, 期望 30（读自 db_max_idle_conn）", mode, got.MaxIdleConn)
		}
		if got.MaxOpenConn != 100 {
			t.Errorf("[%s] MaxOpenConn = %d, 期望 100（读自 db_max_open_conn）", mode, got.MaxOpenConn)
		}
		if got.ConnMaxLifetime != time.Hour {
			t.Errorf("[%s] ConnMaxLifetime = %v, 期望 1h（读自 db_conn_max_lifetime）", mode, got.ConnMaxLifetime)
		}
		if got.ConnMaxIdleTime != 10*time.Minute {
			t.Errorf("[%s] ConnMaxIdleTime = %v, 期望 10m（读自 db_conn_max_idle_time）", mode, got.ConnMaxIdleTime)
		}
	}
}

// TestPoolConfigDefaults 配置缺失时应落到默认值，而不是把 0 传给 database/sql
// （SetMaxOpenConns(0) 的含义是"不限制"，与"没配置"必须区分开）
func TestPoolConfigDefaults(t *testing.T) {
	got := PoolConfig{}.withDefaults()

	if got.MaxIdleConn != defaultMaxIdleConn || got.MaxOpenConn != defaultMaxOpenConn {
		t.Errorf("默认连接数 = %d/%d, 期望 %d/%d",
			got.MaxIdleConn, got.MaxOpenConn, defaultMaxIdleConn, defaultMaxOpenConn)
	}
	if got.ConnMaxLifetime != defaultConnMaxLifetime || got.ConnMaxIdleTime != defaultConnMaxIdleTime {
		t.Errorf("默认存活时长 = %v/%v, 期望 %v/%v",
			got.ConnMaxLifetime, got.ConnMaxIdleTime, defaultConnMaxLifetime, defaultConnMaxIdleTime)
	}
}

// TestPoolConfigIdleClampedToOpen 空闲上限不应超过连接上限
func TestPoolConfigIdleClampedToOpen(t *testing.T) {
	got := PoolConfig{MaxIdleConn: 50, MaxOpenConn: 10}.withDefaults()
	if got.MaxIdleConn != 10 {
		t.Errorf("MaxIdleConn = %d, 期望被压到 MaxOpenConn=10", got.MaxIdleConn)
	}
}
