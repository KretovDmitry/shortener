package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/KretovDmitry/shortener/internal/cfg"
	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/shorturl"
	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
)

type (
	shortenJSONRequestPayload struct {
		URL string `json:"url"`
	}

	shortURL string

	shortenJSONResponsePayload struct {
		Result shortURL `json:"result"`
	}
)

func (s shortURL) MarshalJSON() ([]byte, error) {
	result := fmt.Sprintf("http://%s/%s", cfg.AddrToReturn, s)
	return json.Marshal(result)
}

func (ctx *handlerContext) ShortenJSON(w http.ResponseWriter, r *http.Request) {
	l := logger.Get()
	defer l.Sync()

	// guard in case of future router switching
	if r.Method != http.MethodPost {
		l.Info("got request with bad method", zap.String("method", r.Method))
		msg := `Only POST method is allowed`
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if contentType != "application/json" {
		l.Info(
			"got request with bad content-type",
			zap.String("content-type", r.Header.Get("Content-Type")),
		)
		msg := `Only "application/json" Content-Type is allowed`
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var payload shortenJSONRequestPayload

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	if err := decoder.Decode(&payload); err != nil {
		l.Error("failed decode request JSON body", zap.Error(err))
		msg := fmt.Sprintf(
			"Couldn't decode request JSON body: %s", err,
		)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if len(payload.URL) == 0 {
		msg := "The URL field in the JSON body of the request is empty"
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if !govalidator.IsURL(payload.URL) {
		msg := fmt.Sprintf(
			"The provided string is not a URL: %s", payload.URL,
		)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	sURL, err := shorturl.Generate(payload.URL)
	if err != nil {
		l.Error("failed generate short URL", zap.Error(err))
		msg := fmt.Sprintf("Internal server error: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if err := ctx.store.SaveURL(db.ShortURL(sURL), db.OriginalURL(payload.URL)); err != nil {
		l.Error("failed save URLs", zap.Error(err))
		msg := fmt.Sprintf("Internal server error: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	result := shortenJSONResponsePayload{
		Result: shortURL(sURL),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(result); err != nil {
		l.Error("failed encode response JSON body", zap.Error(err))
		msg := fmt.Sprintf(
			"Couldn't encode response JSON body: %s", err,
		)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}
