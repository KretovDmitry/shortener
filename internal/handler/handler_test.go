package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/repository"
	"github.com/KretovDmitry/shortener/internal/repository/memstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType     = "Content-Type"
	textPlain       = "text/plain; charset=utf-8"
	applicationJSON = "application/json"
)

var errIntentionallyNotWorkingMethod = errors.New("intentionally not working method")

// simulating errors with storage operations.
type brokenStore struct{}

var _ repository.URLStorage = (*brokenStore)(nil)

func (s *brokenStore) Save(context.Context, *models.URL) error {
	return errIntentionallyNotWorkingMethod
}

func (s *brokenStore) SaveAll(context.Context, []*models.URL) error {
	return errIntentionallyNotWorkingMethod
}

func (s *brokenStore) Get(context.Context, models.ShortURL) (*models.URL, error) {
	return nil, errIntentionallyNotWorkingMethod
}

func (s *brokenStore) GetAllByUserID(context.Context, string) ([]*models.URL, error) {
	return nil, errIntentionallyNotWorkingMethod
}

func (s *brokenStore) DeleteURLs(context.Context, ...*models.URL) error {
	return errIntentionallyNotWorkingMethod
}

func (s *brokenStore) Ping(context.Context) error {
	return errIntentionallyNotWorkingMethod
}

type brokenReader struct{}

func (br *brokenReader) Read(_ []byte) (int, error) {
	return 0, errIntentionallyNotWorkingMethod
}

func initMockStore(u *models.URL) *memstore.URLRepository {
	s := memstore.NewURLRepository()
	_ = s.Save(context.TODO(), u)
	return s
}

func TestNew(t *testing.T) {
	type args struct {
		store repository.URLStorage
	}
	tests := []struct {
		args    args
		want    *Handler
		name    string
		wantErr bool
	}{
		{
			name: "positive test #1",
			args: args{
				store: memstore.NewURLRepository(),
			},
			want: &Handler{
				store: memstore.NewURLRepository(),
			},
			wantErr: false,
		},
		{
			name: "negative test #1: nil store",
			args: args{
				store: nil,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, _ := logger.NewForTest()
			got, err := New(tt.args.store, config.NewForTest(), l)
			if !assert.Equal(t, tt.wantErr, err != nil) {
				t.Errorf("Error message: %s\n", err)
			}
			if !tt.wantErr {
				if !assert.Equal(t, got.store, tt.want.store) {
					t.Errorf("got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestIsTextPlainContentType(t *testing.T) {
	testcases := []struct {
		contentType string
		expected    bool
	}{
		{"text/plain", true},
		{"text/plain; charset=utf-8", true},
		{"text/plain; charset=utf-16", true},
		{"application/json; charset=utf-8", false},
		{"application/json; charset=utf-16", false},
	}

	for _, tc := range testcases {
		t.Run(tc.contentType, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			r.Header.Set(contentType, tc.contentType)
			assert.Equal(t, tc.expected, isTextPlainContentType(r))
		})
	}
}

func getResponseTextPayload(t *testing.T, res *http.Response) string {
	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, res.Body.Close(), "failed close body")
	require.NoError(t, err)
	return strings.TrimSpace(string(resBody))
}

func getShortURL(s string) string {
	var res string
	if strings.HasPrefix(s, "http") {
		slice := strings.Split(s, "/")
		res = slice[len(slice)-1]
	}
	return res
}
