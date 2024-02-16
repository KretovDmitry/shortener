// Package handler provides handlers.
package handler

import (
	"errors"
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

type handlerContext struct {
	store db.Storage
}

// NewHandlerContext constructs a new handlerContext,
// ensuring that the dependencies are valid values
func NewHandlerContext(store db.Storage) (*handlerContext, error) {
	if store == nil {
		return nil, errors.New("nil store")
	}
	return &handlerContext{store}, nil
}

func (ctx *handlerContext) CreateShortURL(w http.ResponseWriter, r *http.Request) {
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
	r.Body.Close()

	if len(body) == 0 {
		http.Error(w, "Empty body, must contain URL", http.StatusBadRequest)
		return
	}

	originalURL := string(body)

	shortURL, err := shorturl.Generate(originalURL)
	if err != nil {
		log.Printf("create short URL: %s\n", err)
		http.Error(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}

	if err := ctx.store.SaveURL(shortURL, originalURL); err != nil {
		log.Printf("store save URL: %s\n", err)
		http.Error(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("http://%s/%s", cfg.AddrToReturn, shortURL)))
}

var Base58Regexp = regexp.MustCompile(`^[A-HJ-NP-Za-km-z1-9]{8}$`)

func (ctx *handlerContext) HandleShortURLRedirect(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "shortURL")

	if !Base58Regexp.MatchString(shortURL) {
		http.Error(w, "Invalid URL: "+shortURL, http.StatusBadRequest)
		return
	}

	url, err := ctx.store.RetrieveInitialURL(shortURL)
	if errors.Is(err, db.ErrURLNotFound) {
		http.Error(w, "No such URL: "+shortURL, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
