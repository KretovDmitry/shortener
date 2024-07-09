package myrpc

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	pb "github.com/KretovDmitry/shortener/internal/api/myrpc/proto"
	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/errs"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/models"
	"github.com/KretovDmitry/shortener/internal/models/user"
	"github.com/KretovDmitry/shortener/internal/repository"
	"github.com/KretovDmitry/shortener/internal/shorturl"
	"github.com/asaskevich/govalidator"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ShortenerServer implements all methods of the described prc server.
type ShortenerServer struct {
	// must embed for compatibility.
	pb.UnimplementedShortenerServer
	// store is the database URL storage.
	store repository.URLStorage
	// application configuration.
	config *config.Config
	// logger is the application logger.
	logger logger.Logger
	// deleteURLsChan is a channel for sending deleted URLs to be flushed from the database.
	deleteURLsChan chan *models.URL
	// wg is a wait group used to manage the goroutine that flushes deleted URLs.
	wg *sync.WaitGroup
	// done is a channel used to signal the stop of the handler.
	done chan struct{}
	// bufLen is the buffer length for storing deleted URLs before flushing them to the database.
	bufLen int
}

// Base58Regexp is a regular expression that matches a valid Base58-encoded string.
// It is used to validate the format of shortened URLs.
var Base58Regexp = regexp.MustCompile(`^[A-HJ-NP-Za-km-z1-9]+$`)

// NewServer registers a new server, ensuring that the dependencies are valid values.
func NewServer(
	store repository.URLStorage,
	config *config.Config,
	logger logger.Logger,
) (*ShortenerServer, error) {
	if config == nil {
		return nil, fmt.Errorf("%w: config", errs.ErrNilDependency)
	}
	if config.DeleteBufLen <= 0 {
		return nil, errors.New("buffer length should be >= 1")
	}

	s := &ShortenerServer{
		store:          store,
		config:         config,
		logger:         logger,
		deleteURLsChan: make(chan *models.URL),
		wg:             &sync.WaitGroup{},
		done:           make(chan struct{}),
		bufLen:         config.DeleteBufLen,
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.flushDeletedURLs()
	}()

	return s, nil
}

// Interface implementation gurads.
var _ pb.ShortenerServer = (*ShortenerServer)(nil)

// ShortenURL shortens original url.
func (s *ShortenerServer) ShortenURL(ctx context.Context, in *pb.ShortenURLIn,
) (*pb.ShortenURLOut, error) {
	var out pb.ShortenURLOut
	originalURL := in.GetOriginalUrl()

	// Check if the URL is provided.
	if len(originalURL) == 0 {
		return nil, status.Error(codes.InvalidArgument, "url is not provided")
	}

	// Check if the URL is a valid URL.
	if !govalidator.IsURL(originalURL) {
		return nil, status.Error(codes.InvalidArgument, "invalid url")
	}

	// Generate the shortened URL.
	shortenedURL := shorturl.Generate(originalURL)

	// Extract the user ID from the request context.
	user, ok := user.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no user found")
	}

	// Create a new record with the generated short URL, original URL, and user ID.
	newRecord := models.NewRecord(shortenedURL, originalURL, user.ID)

	// Save the record to the database.
	storeErr := s.store.Save(ctx, newRecord)
	if storeErr != nil {
		if !errors.Is(storeErr, errs.ErrConflict) {
			return nil, status.Error(codes.AlreadyExists, "already exists")
		}
		return nil, status.Error(codes.Internal, "failed to save to database")
	}

	out.ShortUrl = fmt.Sprintf("http://%s/%s", s.config.Server.ReturnAddress, shortenedURL)

	return &out, nil
}

// ShortenBatch shortens urls in batch.
func (s *ShortenerServer) ShortenBatch(ctx context.Context, in *pb.ShortenBatchIn,
) (*pb.ShortenBatchOut, error) {
	var out pb.ShortenBatchOut
	ln := len(in.GetItems())
	recordsToSave := make([]*models.URL, ln)
	items := make([]*pb.ShortenBatchOut_ShortenBatchItemOut, ln)

	user, ok := user.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no user found")
	}

	for i, item := range in.GetItems() {
		originalURL := item.GetOriginalUrl()
		// Check if the URL is provided.
		if len(originalURL) == 0 {
			return nil, status.Error(codes.InvalidArgument, "url is not provided")
		}

		// Check if the URL is a valid URL.
		if !govalidator.IsURL(originalURL) {
			return nil, status.Error(codes.InvalidArgument, "invalid url")
		}

		// Generate the shortened URL.
		shortenedURL := shorturl.Generate(originalURL)
		recordsToSave[i] = models.NewRecord(shortenedURL, originalURL, user.ID)
		shortenedURL = fmt.Sprintf("http://%s/%s", s.config.Server.ReturnAddress, shortenedURL)
		items[i] = &pb.ShortenBatchOut_ShortenBatchItemOut{
			CorrelationId: item.GetCorrelationId(),
			ShortUrl:      shortenedURL,
		}
	}

	out.Items = items

	// Save the records to the database.
	if err := s.store.SaveAll(ctx, recordsToSave); err != nil {
		return nil, status.Error(codes.Internal, "failed to save to database")
	}

	return &out, nil
}

// DeleteURLs deletes all provided urls.
func (s *ShortenerServer) DeleteURLs(ctx context.Context, in *pb.DeleteURLsIn,
) (*pb.DeleteURLsOut, error) {
	var out pb.DeleteURLsOut

	// Extract the user from the context.
	user, ok := user.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no user found")
	}

	// Schedule deletion of the URLs.
	for _, shortURL := range in.GetUrls() {
		s.deleteURLsChan <- &models.URL{
			ShortURL: models.ShortURL(shortURL),
			UserID:   user.ID,
		}
	}

	return &out, nil
}

// GetURLs returns all urls owned by the user.
func (s *ShortenerServer) GetURLs(ctx context.Context, _ *pb.GetURLsIn,
) (*pb.GetURLsOut, error) {
	var out pb.GetURLsOut

	// Extract the user from the context.
	user, ok := user.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no user found")
	}

	urls, err := s.store.GetAllByUserID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "nothing found")
		}
		return nil, status.Error(codes.Internal, "failed to get from DB")
	}

	items := make([]*pb.GetURLsOut_GetURLsOutItem, len(urls))
	for i, u := range urls {
		su := fmt.Sprintf("http://%s/%s", s.config.Server.ReturnAddress, u.ShortURL)
		items[i] = &pb.GetURLsOut_GetURLsOutItem{
			ShortUrl:    su,
			OriginalUrl: string(u.OriginalURL),
		}
	}

	out.Items = items

	return &out, nil
}

// Redirect returns original url corresponding to the provided short one.
func (s *ShortenerServer) Redirect(ctx context.Context, in *pb.RedirectIn,
) (*pb.RedirectOut, error) {
	var out pb.RedirectOut
	shortURL := in.GetShortUrl()

	// Check if shortened url is valid.
	if !Base58Regexp.MatchString(shortURL) {
		return nil, status.Error(codes.InvalidArgument, "invalid url")
	}

	// Get original URL.
	record, err := s.store.Get(ctx, models.ShortURL(shortURL))
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "nothing found")
		}
		return nil, status.Error(codes.Internal, "failed to retrieve url")
	}

	// NotFound again. Is there StatusGone analogue?
	if record.IsDeleted {
		return nil, status.Error(codes.NotFound, "nothing found")
	}

	out.Location = string(record.OriginalURL)

	return &out, nil
}

// Ping checks the status of the database connection.
func (s *ShortenerServer) Ping(ctx context.Context, _ *pb.PingIn,
) (*pb.PingOut, error) {
	var out pb.PingOut
	if err := s.store.Ping(ctx); err != nil {
		if errors.Is(err, errs.ErrDBNotConnected) {
			return nil, status.Error(codes.Unavailable, "DB not connected")
		}
		return nil, status.Error(codes.Internal, "connection error")
	}
	return &out, nil
}

// GetStats reveals total number of users and shortened urls in JSON format.
func (s *ShortenerServer) GetStats(ctx context.Context, _ *pb.GetStatsIn,
) (*pb.GetStatsOut, error) {
	var out pb.GetStatsOut

	count, err := s.store.CountShortURLs(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to count urls")
	}

	out.Urls = int64(count)

	count, err = s.store.CountUsers(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to count users")
	}

	out.Users = int64(count)

	return &out, nil
}

// Stop stops the server and waits for all goroutines to finish.
// It is safe for concurrent use.
func (s *ShortenerServer) Stop() {
	sync.OnceFunc(func() {
		close(s.done)
	})()

	ready := make(chan struct{})
	go func() {
		defer close(ready)
		s.wg.Wait()
	}()

	select {
	case <-time.After(s.config.Server.ShutdownTimeout):
		s.logger.Error("handler stop: shutdown timeout exceeded")
	case <-ready:
		return
	}
}

// flushDeletedURLs is a goroutine that periodically flushes the deleted URLs
// from the buffer to the database. It uses a ticker to trigger the flush
// operation every 10 seconds. If the channel for sending deleted URLs is closed,
// the goroutine stops.
// It is safe for concurrent use.
func (s *ShortenerServer) flushDeletedURLs() {
	ticker := time.NewTicker(10 * time.Second)
	urls := make([]*models.URL, 0, s.bufLen)

	for {
		select {
		case url, open := <-s.deleteURLsChan:
			if !open {
				return
			}
			urls = append(urls, url)

		case <-s.done:
			if len(urls) == 0 {
				return
			}
			_ = s.flush(urls...)
			return

		case <-ticker.C:
			if len(urls) == 0 {
				continue
			}
			if err := s.flush(urls...); err != nil {
				continue
			}
			// reset buffer only when flush succeeded
			urls = urls[:0:s.bufLen]
		}
	}
}

// flush deletes the given URLs from the database.
func (s *ShortenerServer) flush(urls ...*models.URL) error {
	if len(urls) == 0 {
		return nil
	}

	err := s.store.DeleteURLs(context.TODO(), urls...)
	if err != nil {
		s.logger.Error("failed to delete URLs", zap.Error(err),
			zap.Int("num", len(urls)), zap.Any("urls", urls))
	}

	return err
}
