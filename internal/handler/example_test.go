package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/models/user"
)

func Example() {
	// Init handler.
	handler := &Handler{store: db.NewInMemoryStore()}

	// Prepare request and recorder.
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://go.dev/"))
	r.Header.Set(contentType, textPlain)
	r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: "test"}))
	w := httptest.NewRecorder()

	// Make request.
	handler.PostShortenText(w, r)

	// Get results.
	res := w.Result()
	b, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()

	if bytes.HasPrefix(b, []byte("http")) {
		fmt.Println(string(b[bytes.LastIndex(b, []byte("/"))+1:]))
	}

	// Output: YBbxJEcQ
}
