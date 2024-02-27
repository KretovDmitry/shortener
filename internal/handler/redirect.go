package handler

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

var Base58Regexp = regexp.MustCompile(`^[A-HJ-NP-Za-km-z1-9]{8}$`)

func (ctx *handlerContext) HandleShortURLRedirect(w http.ResponseWriter, r *http.Request) {
	l := logger.Get()
	defer l.Sync()

	shortURL := chi.URLParam(r, "shortURL")

	if !Base58Regexp.MatchString(shortURL) {
		l.Info("requested invalid URL", zap.String("url", shortURL))
		http.Error(w, "Invalid URL: "+shortURL, http.StatusBadRequest)
		return
	}

	url, err := ctx.store.RetrieveInitialURL(shortURL)
	if errors.Is(err, db.ErrURLNotFound) {
		l.Info("requested non-existent URL", zap.String("url", shortURL))
		http.Error(w, "No such URL: "+shortURL, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
