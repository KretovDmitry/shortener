// Package handler provides handlers.
package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/KretovDmitry/shortener/internal/cfg"
	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/shorturl"
	"github.com/go-chi/chi/v5"
)

func CreateShortURL(w http.ResponseWriter, r *http.Request) {
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if i := strings.Index(contentType, ";"); i > -1 {
		contentType = contentType[0:i]
	}
	if contentType != "text/plain" {
		msg := `Only "text/plain" Content-Type is allowed`
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("create short URL failed to read body: %s\n", err)
		http.Error(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}

	if len(body) == 0 {
		http.Error(w, "Empty body, must contain URL", http.StatusBadRequest)
		return
	}

	originalURL := string(body)

	shortURL, err := shorturl.Generate(originalURL)
	if err != nil {
		log.Printf("create short URL: %s\n", err)
		http.Error(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
	}

	db.SaveURL(shortURL, originalURL)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	fmt.Println(cfg.AddrToReturn)
	w.Write([]byte(fmt.Sprintf("http://%s/%s", cfg.AddrToReturn, shortURL)))
}

var Base58Regexp = regexp.MustCompile(`^[A-HJ-NP-Za-km-z1-9]{8}$`)

func HandleShortURLRedirect(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "shortURL")

	if !Base58Regexp.MatchString(shortURL) {
		http.Error(w, "Invalid URL: "+shortURL, http.StatusBadRequest)
		return
	}

	url, found := db.RetrieveInitialURL(shortURL)

	if !found {
		http.Error(w, "No such URL: "+shortURL, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
