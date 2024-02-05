// Package handler provides handlers.
package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/shorturl"
)

var HomeRegexp = regexp.MustCompile(`^\/$`)

func CreateShortURL(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("create short URL failed to read body: %s\n", err)
		http.Error(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}

	originalURL := string(body)
	shortURL, err := shorturl.GenerateShortLink(originalURL)
	if err != nil {
		log.Printf("create short URL: %s\n", err)
		http.Error(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
	}

	db.SaveURLMapping(shortURL, originalURL)

	resp := "http://" + r.Host + "/" + shortURL

	w.Header().Set("content-type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(resp))
}

var Base58Regexp = regexp.MustCompile(`^\/[A-HJ-NP-Za-km-z1-9]{8}$`)

func HandleShortURLRedirect(w http.ResponseWriter, r *http.Request) {
	if !Base58Regexp.MatchString(r.URL.Path) {
		msg := fmt.Sprintf("Specified URL does not meet the requirements of the service: %s\n", r.URL.Path[1:])
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	url, found := db.RetrieveInitialURL(r.URL.Path[1:])
	if !found {
		msg := fmt.Sprintf("No such short URL: %s\n", r.URL.Path[1:])
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	w.Header().Set("location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
