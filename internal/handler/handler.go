package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/KretovDmitry/shortener/internal/cfg"
	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/shorturl"
	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
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
	return &handlerContext{
		store: store,
	}, nil
}

func (ctx *handlerContext) ShortenText(w http.ResponseWriter, r *http.Request) {
	l := logger.Get()
	defer l.Sync()

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
		l.Error("failed to read request body", zap.Error(err))
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
		l.Error("failed to generate short URL", zap.Error(err))
		http.Error(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}

	if err := ctx.store.SaveURL(shortURL, originalURL); err != nil {
		l.Error("failed to save URL", zap.Error(err))
		http.Error(w, fmt.Sprintf("Internal server error: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("http://%s/%s", cfg.AddrToReturn, shortURL)))
}

type shortenJSONRequestPayload struct {
	URL string `json:"url,omitempty"`
}

type (
	shortURL string

	shortenJSONResponsePayload struct {
		Result shortURL `json:"result,omitempty"`
	}
)

func (s shortURL) MarshalJSON() ([]byte, error) {
	result := fmt.Sprintf("http://%s/%s", cfg.AddrToReturn, s)
	return json.Marshal(result)
}

func (ctx *handlerContext) ShortenJSON(w http.ResponseWriter, r *http.Request) {
	l := logger.Get()
	defer l.Sync()

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
		l.Error("cannot decode request JSON body", zap.Error(err))
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
		l.Error("failed to generate short URL", zap.Error(err))
		msg := fmt.Sprintf("Internal server error: %s", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	if err := ctx.store.SaveURL(sURL, payload.URL); err != nil {
		l.Error("failed to save URL", zap.Error(err))
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
		l.Error("cannot encode response JSON body", zap.Error(err))
		msg := fmt.Sprintf(
			"Couldn't encode request JSON body: %s", err,
		)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
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
