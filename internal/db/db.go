// Package db is responsible for storing short URLs.
package db

import (
	"sync"

	"github.com/KretovDmitry/shortener/internal/shorturl"
)

type storage interface {
	get(key string) (url string, found bool)
	set(url string) (string, error)
}

type memo struct {
	mu    sync.RWMutex
	cache map[string]string
}

func (memo *memo) get(key string) (url string, found bool) {
	memo.mu.RLock()
	defer memo.mu.RUnlock()
	url, found = memo.cache[key]
	return
}

func (memo *memo) set(url string) (string, error) {
	memo.mu.Lock()
	defer memo.mu.Unlock()
	shortUrl, err := shorturl.GenerateShortLink(url)
	if err != nil {
		return "", err
	}
	memo.cache[shortUrl] = url
	return shortUrl, nil
}

var m storage = &memo{cache: make(map[string]string)}

func RetrieveInitialUrl(shortUrl string) (ulr string, found bool) {
	return m.get(shortUrl)
}

func SaveUrlMapping(url string) (string, error) {
	return m.set(url)
}
