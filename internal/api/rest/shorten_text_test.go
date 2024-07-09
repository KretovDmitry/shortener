package rest

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"github.com/KretovDmitry/shortener/internal/repository/memstore"
	"github.com/KretovDmitry/shortener/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestPostShortenText_NewRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	m := mocks.NewMockURLStorage(ctrl)

	userID := "test"

	testcases := []string{
		"http://foo.bar#com",
		"http://foobar.com",
		"https://foobar.com",
		"foobar.com",
		"http://foobar.coffee/",
		"http://foobar.中文网/",
		"http://foobar.org/",
		"http://foobar.ORG",
		"http://foobar.org:8080/",
		"ftp://foobar.ru/",
		"ftp.foo.bar",
		"http://user:pass@www.foobar.com/",
		"http://user:pass@www.foobar.com/path/file",
		"http://127.0.0.1/",
		"http://duckduckgo.com/?q=%2F",
		"http://localhost:3000/",
		"http://foobar.com/?foo=bar#baz=qux",
		"http://foobar.com?foo=bar",
		"http://www.xn--froschgrn-x9a.net/",
		"http://foobar.com/a-",
		"http://foobar.پاکستان/",
		"http://foo_bar.com",
		"http://user:pass@foo_bar_bar.bar_foo.com",
		"http://localhost:3000/",
		"http://foobar.com#baz=qux",
		"http://foobar.com/t$-_.+!*\\'(),",
		"http://www.foobar.com/~foobar",
		"http://r6---snnvoxuioq6.googlevideo.com",
		"mailto:someone@example.com",
		"irc://#channel@network",
		"http://foo.bar.org",
		"http://www.foo.bar.org",
		"http://www.foo.co.uk",
		"http://myservice.:9093/",
		"https://pbs.twimg.com/profile_images/560826135676588032/j8fWrmYY_normal.jpeg",
		"http://prometheus-alertmanager.service.q:9093",
		"aio1_alertmanager_container-63376c45:9093",
		"https://www.logn-123-123.url.with.sigle.letter.d:12345/url/path/foo?bar=zzz#user",
		"http://me.example.com",
		"http://www.me.example.com",
		"https://farm6.static.flickr.com",
		"https://zh.wikipedia.org/wiki/Wikipedia:%E9%A6%96%E9%A1%B5",
		"http://hyphenated-host-name.example.co.in",
		"http://www.domain-can-have-dashes.com",
		"http://m.abcd.com/test.html",
		"http://m.abcd.com/a/b/c/d/test.html?args=a&b=c",
		"http://[::1]:9093",
		"http://[2001:db8:a0b:12f0::1]/index.html",
		"http://[1200:0000:AB00:1234:0000:2552:7777:1313]",
		"http://user:pass@[::1]:9093/a/b/c/?a=v#abc",
		"https://127.0.0.1/a/b/c?a=v&c=11d",
		"https://foo_bar.example.com",
		"http://foo_bar.example.com",
		"http://foo_bar_fizz_buzz.example.com",
		"foo_bar.example.com",
		"foo_bar_fizz_buzz.example.com",
		"http://hello_world.example.com",
		"foo_bar-fizz-buzz:1313",
	}
	for _, tc := range testcases {
		t.Run(tc, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc))
			r.Header.Set(contentType, textPlain)
			r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: userID}))

			w := httptest.NewRecorder()

			m.EXPECT().
				Save(gomock.Any(), gomock.Any()).
				Times(1).
				Return(nil)

			l, _ := logger.NewForTest()
			c := config.NewForTest()

			handler, err := NewHandler(m, c, l)
			require.NoError(t, err, "failed to init handler")

			handler.PostShortenText(w, r)

			res := w.Result()
			require.NoError(t, res.Body.Close(), "failed close body")

			assert.Equal(t, http.StatusCreated, res.StatusCode)
			assert.Equal(t, textPlain, res.Header.Get(contentType))
		})
	}
}

func TestPostShortenText_RepeatedRecord(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	m := mocks.NewMockURLStorage(ctrl)

	userID := "test"

	testcases := []string{
		"http://foo.bar#com",
		"http://foobar.com",
		"https://foobar.com",
		"foobar.com",
		"http://foobar.coffee/",
		"http://foobar.中文网/",
		"http://foobar.org/",
		"http://foobar.ORG",
		"http://foobar.org:8080/",
		"ftp://foobar.ru/",
		"ftp.foo.bar",
		"http://user:pass@www.foobar.com/",
		"http://user:pass@www.foobar.com/path/file",
		"http://127.0.0.1/",
		"http://duckduckgo.com/?q=%2F",
		"http://localhost:3000/",
		"http://foobar.com/?foo=bar#baz=qux",
		"http://foobar.com?foo=bar",
		"http://www.xn--froschgrn-x9a.net/",
		"http://foobar.com/a-",
		"http://foobar.پاکستان/",
		"http://foo_bar.com",
		"http://user:pass@foo_bar_bar.bar_foo.com",
		"http://localhost:3000/",
		"http://foobar.com#baz=qux",
		"http://foobar.com/t$-_.+!*\\'(),",
		"http://www.foobar.com/~foobar",
		"http://r6---snnvoxuioq6.googlevideo.com",
		"mailto:someone@example.com",
		"irc://#channel@network",
		"http://foo.bar.org",
		"http://www.foo.bar.org",
		"http://www.foo.co.uk",
		"http://myservice.:9093/",
		"https://pbs.twimg.com/profile_images/560826135676588032/j8fWrmYY_normal.jpeg",
		"http://prometheus-alertmanager.service.q:9093",
		"aio1_alertmanager_container-63376c45:9093",
		"https://www.logn-123-123.url.with.sigle.letter.d:12345/url/path/foo?bar=zzz#user",
		"http://me.example.com",
		"http://www.me.example.com",
		"https://farm6.static.flickr.com",
		"https://zh.wikipedia.org/wiki/Wikipedia:%E9%A6%96%E9%A1%B5",
		"http://hyphenated-host-name.example.co.in",
		"http://www.domain-can-have-dashes.com",
		"http://m.abcd.com/test.html",
		"http://m.abcd.com/a/b/c/d/test.html?args=a&b=c",
		"http://[::1]:9093",
		"http://[2001:db8:a0b:12f0::1]/index.html",
		"http://[1200:0000:AB00:1234:0000:2552:7777:1313]",
		"http://user:pass@[::1]:9093/a/b/c/?a=v#abc",
		"https://127.0.0.1/a/b/c?a=v&c=11d",
		"https://foo_bar.example.com",
		"http://foo_bar.example.com",
		"http://foo_bar_fizz_buzz.example.com",
		"foo_bar.example.com",
		"foo_bar_fizz_buzz.example.com",
		"http://hello_world.example.com",
		"foo_bar-fizz-buzz:1313",
	}

	for _, tc := range testcases {
		t.Run(tc, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc))
			r.Header.Set(contentType, textPlain)
			r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: userID}))

			w := httptest.NewRecorder()

			m.EXPECT().
				Save(gomock.Any(), gomock.Any()).
				Times(1).
				Return(errs.ErrConflict)

			l, _ := logger.NewForTest()
			c := config.NewForTest()

			handler, err := NewHandler(m, c, l)
			require.NoError(t, err, "failed to init handler")

			handler.PostShortenText(w, r)

			res := w.Result()
			require.NoError(t, res.Body.Close(), "failed close body")

			assert.Equal(t, http.StatusConflict, res.StatusCode)
			assert.Equal(t, textPlain, res.Header.Get(contentType))
		})
	}
}

func TestPostShortenText_BadMethods(t *testing.T) {
	t.Parallel()
	path := "/"
	payload := "https://go.dev"

	tests := []struct {
		name   string
		method string
	}{
		{"invalid method: get", http.MethodGet},
		{"invalid method: put", http.MethodPut},
		{"invalid method: head", http.MethodHead},
		{"invalid method: patch", http.MethodPatch},
		{"invalid method: trace", http.MethodTrace},
		{"invalid method: delete", http.MethodDelete},
		{"invalid method: connect", http.MethodConnect},
		{"invalid method: options", http.MethodOptions},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(tt.method, path, strings.NewReader(payload))
			w := httptest.NewRecorder()

			l, _ := logger.NewForTest()
			c := config.NewForTest()

			handler, err := NewHandler(memstore.NewURLRepository(), c, l)
			require.NoError(t, err, "new handler context error")

			handler.PostShortenText(w, r)

			res := w.Result()

			response := getResponseTextPayload(t, res)
			require.NoError(t, res.Body.Close(), "failed close body")

			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
			assert.Equal(t, textPlain, res.Header.Get(contentType))
			assert.Equal(t,
				fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, tt.method),
				response,
			)
		})
	}
}

func TestPostShortenText_BadContentTypes(t *testing.T) {
	t.Parallel()
	path := "/"
	payload := "https://go.dev"

	contentTypes := []string{
		"application/java-archive",
		"application/EDI-X12",
		"application/EDIFACT",
		"application/javascript (obsolete)",
		"application/octet-stream",
		"application/ogg",
		"application/pdf",
		"application/xhtml+xml",
		"application/x-shockwave-flash",
		"application/json",
		"application/ld+json",
		"application/xml",
		"application/zip",
		"application/x-www-form-urlencoded",
		"audio/mpeg",
		"audio/x-ms-wma",
		"audio/vnd.rn-realaudio",
		"audio/x-wav",
		"image/gif",
		"image/jpeg",
		"image/png",
		"image/tiff",
		"image/vnd.microsoft.icon",
		"image/x-icon",
		"image/vnd.djvu",
		"image/svg+xml",
		"multipart/mixed",
		"multipart/alternative",
		"multipart/related",
		"multipart/form-data",
		"text/css",
		"text/csv",
		"text/html",
		"text/javascript",
		"text/xml",
		"video/mpeg",
		"video/mp4",
		"video/quicktime",
		"video/x-ms-wmv",
		"video/x-msvideo",
		"video/x-flv",
		"video/webm",
		"application/vnd.android.package-archive",
		"application/vnd.oasis.opendocument.text",
		"application/vnd.oasis.opendocument.spreadsheet",
		"application/vnd.oasis.opendocument.presentation",
		"application/vnd.oasis.opendocument.graphics",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.mozilla.xul+xml",
	}
	for _, ct := range contentTypes {
		ct := ct
		t.Run(ct, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
			r.Header.Set(contentType, ct)
			w := httptest.NewRecorder()

			l, _ := logger.NewForTest()
			c := config.NewForTest()

			handler, err := NewHandler(memstore.NewURLRepository(), c, l)
			require.NoError(t, err, "failed to init new handler")

			handler.PostShortenText(w, r)

			res := w.Result()

			response := getResponseTextPayload(t, res)
			require.NoError(t, res.Body.Close(), "failed to close body")

			assert.Equal(t, http.StatusBadRequest, res.StatusCode)
			assert.Equal(t, textPlain, res.Header.Get(contentType))
			assert.Equal(t,
				fmt.Sprintf("%s: %s", errs.ErrInvalidRequest, ct),
				response,
			)
		})
	}
}

func TestPostShortenText_BadReader(t *testing.T) {
	brokenReader := &brokenReader{}

	r := httptest.NewRequest(http.MethodPost, "/", brokenReader)
	r.Header.Set(contentType, textPlain)
	r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: "test"}))

	w := httptest.NewRecorder()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	m := mocks.NewMockURLStorage(ctrl)

	l, _ := logger.NewForTest()
	c := config.NewForTest()

	handler, err := NewHandler(m, c, l)
	require.NoError(t, err, "failed to init new handler")

	handler.PostShortenText(w, r)

	res := w.Result()

	response := getResponseTextPayload(t, res)
	require.NoError(t, res.Body.Close(), "failed to close body")

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode,
		"status code mismatch")
	assert.Equal(t, textPlain, res.Header.Get(contentType))
	assert.Equal(t, fmt.Sprintf(
		"%s: failed to read request body", errIntentionallyNotWorkingMethod,
	), response, "response message mismatch")
}

func TestPostShortenText_BadPayload(t *testing.T) {
	t.Parallel()
	tests := []string{
		"",
		"https://test...com",
		"htps://google.com",
		"htp://do.dev",
		"http;//yandex.ru",
		"https;//mail.ru",
		"https:/yahoo.com",
		"http://foobar.c_o_m",
		"http://_foobar.com",
		"xyz://foobar.com",
		".com",
		"rtmp://foobar.com",
		"http://www.-foobar.com/",
		"http://www.foo---bar.com/",
		"irc://irc.server.org/channel",
		"/abs/test/dir",
		"./rel/test/dir",
		"http://foo^bar.org",
		"http://foo&*bar.org",
		"http://foo&bar.org",
		"http://foo bar.org",
		"foo",
		"http://.foo.com",
		"http://,foo.com",
		",foo.com",
		"google",
		"http://cant-end-with-hyphen-.example.com",
		"http://-cant-start-with-hyphen.example.com",
		"http://[::1]:909388",
		"1200::AB00:1234::2552:7777:1313",
		"http://_cant_start_with_underescore",
		"http://cant_end_with_underescore_",
		"foo_bar-fizz-buzz:13:13",
		"foo_bar-fizz-buzz://1313",
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt))
			r.Header.Set(contentType, textPlain)
			r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: "test"}))

			w := httptest.NewRecorder()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			m := mocks.NewMockURLStorage(ctrl)

			l, _ := logger.NewForTest()
			c := config.NewForTest()

			handler, err := NewHandler(m, c, l)
			require.NoError(t, err, "failed to init new handler")

			handler.PostShortenText(w, r)

			res := w.Result()

			response := getResponseTextPayload(t, res)
			require.NoError(t, res.Body.Close(), "failed close body")

			assert.Equal(t, http.StatusBadRequest, res.StatusCode,
				"status code mismatch")
			assert.Equal(t, textPlain, res.Header.Get(contentType),
				"content type mismatch")
			assert.True(t,
				strings.Contains(response, errs.ErrInvalidRequest.Error()),
				"response message mismatch")
		})
	}
}

func TestPostShortenText_WithoutUserInContext(t *testing.T) {
	path := "/"
	payload := "https://go.dev"

	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
	r.Header.Set(contentType, textPlain)

	w := httptest.NewRecorder()

	l, _ := logger.NewForTest()
	c := config.NewForTest()

	handler, err := NewHandler(memstore.NewURLRepository(), c, l)
	require.NoError(t, err, "failed to init new handler")

	handler.PostShortenText(w, r)

	res := w.Result()

	response := getResponseTextPayload(t, res)
	require.NoError(t, res.Body.Close(), "failed to close body")

	assert.Equal(t, http.StatusUnauthorized, res.StatusCode, "status code mismatch")
	assert.Equal(t, fmt.Sprintf("%s: no user found", errs.ErrUnauthorized),
		response, "response message mismatch")
}

func TestPostShortenText_BadStore(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://go.dev"))
	r.Header.Set(contentType, textPlain)
	r = r.WithContext(user.NewContext(r.Context(), &user.User{ID: "test"}))

	w := httptest.NewRecorder()

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	m := mocks.NewMockURLStorage(ctrl)
	m.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		Times(1).
		Return(errIntentionallyNotWorkingMethod)

	l, _ := logger.NewForTest()
	c := config.NewForTest()

	handler, err := NewHandler(m, c, l)
	require.NoError(t, err, "failed to init new handler")

	handler.PostShortenText(w, r)

	res := w.Result()

	response := getResponseTextPayload(t, res)
	require.NoError(t, res.Body.Close(), "failed to close body")

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode,
		"status code mismatch")
	assert.Equal(t, textPlain, res.Header.Get(contentType))
	assert.Equal(t, fmt.Sprintf(
		"%s: failed to save to database", errIntentionallyNotWorkingMethod,
	), response, "response message mismatch")
}
