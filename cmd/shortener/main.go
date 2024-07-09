package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	myrpc "github.com/KretovDmitry/shortener/internal/api/myrpc"
	pb "github.com/KretovDmitry/shortener/internal/api/myrpc/proto"
	"github.com/KretovDmitry/shortener/internal/api/rest"
	"github.com/KretovDmitry/shortener/internal/config"
	"github.com/KretovDmitry/shortener/internal/logger"
	"github.com/KretovDmitry/shortener/internal/middleware"
	"github.com/KretovDmitry/shortener/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/acme/autocert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Server run context.
	serverCtx, serverStopCtx := context.WithCancel(context.Background())
	defer serverStopCtx()

	// Load application configuration.
	cfg := config.MustLoad()

	// Create root logger tagged with server version.
	logger := logger.New(cfg).With(serverCtx, "version", buildVersion)
	defer func() {
		_ = logger.Sync()
	}()

	// Init URL repository.
	store, err := repository.NewURLStore(cfg, logger)
	if err != nil {
		return fmt.Errorf("failed to init store: %w", err)
	}

	if cfg.RPCEnabled {
		listen, err := net.Listen("tcp", cfg.Server.RunAddress.String())
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}

		// Init new shortener server.
		s, err := myrpc.NewServer(store, cfg, logger)
		if err != nil {
			return fmt.Errorf("filed to init RPC server: %w", err)
		}
		// Stop async short URL deletion.
		defer s.Stop()

		// Register server with interceptors.
		server := grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				logging.UnaryServerInterceptor(logger.InterceptorLogger()),
				recovery.UnaryServerInterceptor(),
				middleware.AuthorizationRPC(cfg, logger),
			),
		)

		pb.RegisterShortenerServer(server, s)
		// for grpcurl testing.
		reflection.Register(server)

		// Graceful shutdown.
		go func() {
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT,
				syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)

			signal := <-sig

			logger.With(serverCtx, "signal", signal.String()).
				Infof("Shutting down server with %s timeout",
					cfg.Server.ShutdownTimeout)

			server.GracefulStop()
			serverStopCtx()
		}()

		logger.Infof("RPC server has started: %s", cfg.Server.RunAddress)
		logger.Infof("Return address: %s", cfg.Server.ReturnAddress)
		if err = server.Serve(listen); err != nil {
			return fmt.Errorf("run server failed: %w", err)
		}
	} else {
		// Init HTTP handlers.
		handler, err := rest.NewHandler(store, cfg, logger)
		if err != nil {
			return fmt.Errorf("new handler: %w", err)
		}
		// Stop async short URL deletion.
		defer handler.Stop()

		// Init HTTP server.
		hs := &http.Server{
			Addr:              cfg.Server.RunAddress.String(),
			ReadHeaderTimeout: cfg.Server.Timeout,
			IdleTimeout:       cfg.Server.IdleTimeout,
			Handler:           handler.Register(chi.NewRouter(), cfg, logger),
		}

		// Graceful shutdown.
		go func() {
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT,
				syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)

			signal := <-sig

			logger.With(serverCtx, "signal", signal.String()).
				Infof("Shutting down server with %s timeout",
					cfg.Server.ShutdownTimeout)

			if err = hs.Shutdown(serverCtx); err != nil {
				logger.Errorf("graceful shutdown failed: %s", err)
			}
			serverStopCtx()
		}()

		logger.Infof("HTTP server has started: %s", cfg.Server.RunAddress)
		logger.Infof("Return address: %s", cfg.Server.ReturnAddress)
		if cfg.TLSEnabled {
			cm := &autocert.Manager{
				Cache:  autocert.DirCache("cache/certs"),
				Prompt: autocert.AcceptTOS,
			}
			hs.TLSConfig = cm.TLSConfig()
			logger.Info("The server is running over the SSL protocol")
			if err = hs.ListenAndServeTLS("", ""); err != nil &&
				!errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("run server failed: %w", err)
			}
		} else {
			if err = hs.ListenAndServe(); err != nil &&
				!errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("run server failed: %w", err)
			}
		}
	}

	// Wait for server context to be stopped
	select {
	case <-serverCtx.Done():
	case <-time.After(cfg.Server.ShutdownTimeout):
		return errors.New("graceful shutdown timed out... forcing exit")
	}

	return nil
}

func printBuildInfo() {
	if buildVersion == "" {
		fmt.Println("Build version: N/A")
	} else {
		fmt.Printf("Build version: %s\n", buildVersion)
	}
	if buildDate == "" {
		fmt.Println("Build date: N/A")
	} else {
		fmt.Printf("Build date: %s\n", buildDate)
	}
	if buildCommit == "" {
		fmt.Println("Build commit: N/A")
	} else {
		fmt.Printf("Build commit: %s\n", buildCommit)
	}
}
