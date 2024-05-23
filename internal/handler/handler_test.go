package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/KretovDmitry/shortener/internal/db"
	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType     = "Content-Type"
	textPlain       = "text/plain; charset=utf-8"
	applicationJSON = "application/json"
)

var (
	errIntentionallyNotWorkingMethod = errors.New("intentionally not working method")
	emptyMockStore                   = initMockStore(&models.URL{OriginalURL: ""})
)

var _ db.URLStorage = (*mockStore)(nil)

type mockStore struct {
	db.URLStorage
}

func initMockStore(u *models.URL) *mockStore {
	inMem := db.NewInMemoryStore()
	_ = inMem.Save(context.TODO(), u)
	return &mockStore{inMem}
}

// simulating errors with storage operations
type brokenStore struct{}

func (s *brokenStore) Save(context.Context, *models.URL) error {
	return errIntentionallyNotWorkingMethod
}

func (s *brokenStore) SaveAll(context.Context, []*models.URL) error {
	return errIntentionallyNotWorkingMethod
}

func (s *brokenStore) Get(context.Context, models.ShortURL) (*models.URL, error) {
	return nil, errIntentionallyNotWorkingMethod
}

func (s *brokenStore) GetAllByUserID(_ context.Context, userID string) ([]*models.URL, error) {
	return nil, nil
}

func (s *brokenStore) DeleteURLs(_ context.Context, urls ...*models.URL) error {
	return nil
}

func (s *brokenStore) Ping(context.Context) error {
	return errIntentionallyNotWorkingMethod
}

type notConnectedStore struct {
	brokenStore
}

func (s *notConnectedStore) Ping(context.Context) error {
	return errs.ErrDBNotConnected
}

func TestNew(t *testing.T) {
	type args struct {
		store db.URLStorage
	}
	tests := []struct {
		name    string
		args    args
		want    *Handler
		wantErr bool
	}{
		{
			name: "positive test #1",
			args: args{
				store: emptyMockStore,
			},
			want: &Handler{
				store: emptyMockStore,
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
			got, err := New(tt.args.store, 5)
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
