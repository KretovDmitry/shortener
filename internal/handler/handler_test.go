package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType     = "Content-Type"
	textPlain       = "text/plain; charset=utf-8"
	applicationJSON = "application/json"
)

var errIntentionallyNotWorkingMethod = errors.New("intentionally not working method")
var emptyMockStore = &mockStore{expectedData: ""}

// mokeStore must implement URLStore interface
type mockStore struct {
	expectedData string
}

// do nothing on create, return ErrConflict if URL already exists
func (s *mockStore) Save(ctx context.Context, url *db.URL) error {
	if s.expectedData == string(url.OriginalURL) {
		return db.ErrConflict
	}
	return nil
}

func (s *mockStore) SaveAll(context.Context, []*db.URL) error {
	return nil
}

// return expected data
func (s *mockStore) Get(context.Context, db.ShortURL) (*db.URL, error) {
	// mock not found error
	if s.expectedData == "" {
		return nil, db.ErrURLNotFound
	}
	return &db.URL{OriginalURL: db.OriginalURL(s.expectedData)}, nil
}
func (s *mockStore) Ping(context.Context) error {
	return nil
}

// simulating errors with storage operations
type brokenStore struct {
	mockStore
}

func (s *brokenStore) Save(context.Context, *db.URL) error {
	return errIntentionallyNotWorkingMethod
}

func (s *brokenStore) Get(context.Context, db.ShortURL) (*db.URL, error) {
	return nil, errIntentionallyNotWorkingMethod
}

func TestNew(t *testing.T) {
	emptyMockStore := &mockStore{expectedData: ""}

	type args struct {
		store db.URLStorage
	}
	tests := []struct {
		name    string
		args    args
		want    *handler
		wantErr bool
	}{
		{
			name: "positive test #1",
			args: args{
				store: emptyMockStore,
			},
			want: &handler{
				store:  emptyMockStore,
				logger: logger.Get(),
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
			got, err := New(tt.args.store)
			if !assert.Equal(t, tt.wantErr, err != nil) {
				t.Errorf("Error message: %s\n", err)
			}
			if !assert.Equal(t, got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

func getResponseTextPayload(t *testing.T, res *http.Response) string {
	resBody, err := io.ReadAll(res.Body)
	res.Body.Close()
	require.NoError(t, err)
	return strings.TrimSpace(string(resBody))
}

func getShortURL(s string) (res string) {
	if strings.HasPrefix(s, "http") {
		slice := strings.Split(s, "/")
		res = slice[len(slice)-1]
	}
	return
}
