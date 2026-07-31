// Package cache 提供带过期时间的内存缓存，替代原先使用的 beego/cache。
// 只实现项目实际用到的 Put / Get / IsExist 三个操作。
package cache

import (
	"sync"
	"time"
)

type item struct {
	value  interface{}
	expire time.Time // 零值表示永不过期
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]item
}

func New() *Cache {
	return &Cache{items: map[string]item{}}
}

// Put 写入缓存，ttl 为 0 表示不过期
func (c *Cache) Put(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	it := item{value: value}
	if ttl > 0 {
		it.expire = time.Now().Add(ttl)
	}
	c.items[key] = it
}

// Get 读取缓存，不存在或已过期返回 nil
func (c *Cache) Get(key string) interface{} {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return nil
	}
	if !it.expire.IsZero() && time.Now().After(it.expire) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil
	}
	return it.value
}

// IsExist 判断键是否存在且未过期
func (c *Cache) IsExist(key string) bool {
	return c.Get(key) != nil
}

// Delete 删除指定键
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}
