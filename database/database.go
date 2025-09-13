package database

import "sync"

// Database 数据存储
type Database struct {
	data sync.Map
}

// NewDatabase 创建新数据库实例
func NewDatabase() *Database {
	return &Database{}
}

// Set 设置键值对
func (db *Database) Set(key, value string) {
	db.data.Store(key, value)
}

// Get 获取值
func (db *Database) Get(key string) (string, bool) {
	v, ok := db.data.Load(key)
	if !ok {
		return "", false
	}
	return v.(string), true
}

// Delete 删除键
func (db *Database) Delete(key string) {
	db.data.Delete(key)
}
