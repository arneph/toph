package db

import "sync"

type DB struct {
	entries map[string]string
	sync.RWMutex
}

func NewDB() *DB {
	db := new(DB)
	db.entries = make(map[string]string)
	return db
}

func (db *DB) Read(key string) string {
	db.RLock()
	defer db.RUnlock()
	value, ok := db.entries[key]
	if !ok {
		panic("key not found")
	}
	return value
}

func (db *DB) Write(key, value string) {
	db.Lock()
	defer db.Unlock()
	db.entries[key] = value
}

func (db *DB) WriteIfAbsent(key, value string) {
	db.RLock() // not always matching RUnlock
	_, ok := db.entries[key]
	if ok {
		db.RUnlock()
		return
	}
	db.Lock() // read lock can not be promoted to write lock
	defer db.Unlock()
	db.entries[key] = value
}
