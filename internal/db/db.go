// Package db is responsible for storing original and short URLs.
package db

import (
	"sync"
)

type storage interface {
	get(key string) (url string, found bool)
	set(shortURL, url string)
}

type db struct {
	mu    sync.RWMutex
	store map[string]string
}

func (d *db) get(key string) (url string, found bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	url, found = d.store[key]
	return
}

func (d *db) set(shortURL, url string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.store[shortURL] = url
}

var m storage = &db{store: make(map[string]string)}

func RetrieveInitialURL(shortURL string) (ulr string, found bool) {
	return m.get(shortURL)
}

func SaveURL(shortURL, url string) {
	m.set(shortURL, url)
}
