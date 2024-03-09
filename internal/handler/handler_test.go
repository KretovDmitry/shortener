package handler

import (
	"io"
	"net/http"
	"reflect"
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

// mokeStore must implement URLStore interface
type mockStore struct {
	expectedData string
}

// do nothing on create
func (s *mockStore) SaveURL(db.ShortURL, db.OriginalURL) error {
	return nil
}

// return expected data
func (s *mockStore) RetrieveInitialURL(db.ShortURL) (db.OriginalURL, error) {
	// mock not found error
	if s.expectedData == "" {
		return "", db.ErrURLNotFound
	}
	return db.OriginalURL(s.expectedData), nil
}

func TestNew(t *testing.T) {
	emptyMockStore := &mockStore{expectedData: ""}

	type args struct {
		store    db.Store
		sqlStore db.Store
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
				store:    emptyMockStore,
				sqlStore: emptyMockStore,
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
				store:    nil,
				sqlStore: emptyMockStore,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.args.store, nil)
			if !assert.Equal(t, tt.wantErr, err != nil) {
				t.Errorf("Error message: %s\n", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

func getTextPayload(t *testing.T, res *http.Response) string {
	resBody, err := io.ReadAll(res.Body)
	defer res.Body.Close()
	require.NoError(t, err)

	return strings.TrimSpace(string(resBody))
}
