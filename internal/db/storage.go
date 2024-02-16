// Package db is responsible for storing original and short URLs.
package db

type Storage interface {
	RetrieveInitialURL(key string) (url string, err error)
	SaveURL(shortURL, url string) error
}
