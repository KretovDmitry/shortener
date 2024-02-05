// Package db is responsible for storing original and short URLs.
package db

import (
	"sync"
)

type DB struct {
	mu    sync.RWMutex
	store map[string]string
}

func (db *DB) get(key string) (url string, found bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	url, found = db.store[key]
	return
}

func (db *DB) set(shortURL, url string) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.store[shortURL] = url
}

// Since I do not know any architectural patterns,
// the main reason I created the next function
// is to be able to provide some mock data for testing

// Init allows to set some initial data
// NOTE: since it is exported, it can be used anywhere in the code
// to fill the storage with garbage data :(
func (db *DB) Init(data map[string]string) *DB {
	db.mu.Lock()
	defer db.mu.Unlock()
	for k, v := range data {
		db.store[k] = v
	}
	return db
}

// global storage
var db *DB

// GetDB returns global storage
func GetDB() *DB {
	if db != nil {
		return db
	}

	db = &DB{store: make(map[string]string)}

	return db
}

// Returns the original URL address, the key is its shortened version
func RetrieveInitialURL(shortURL string) (ulr string, found bool) {
	return GetDB().get(shortURL)
}

// Saves both original and short URL to the database
func SaveURLMapping(shortURL, url string) {
	GetDB().set(shortURL, url)
}
