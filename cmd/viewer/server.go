package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Z3-N0/flexlog"
	"github.com/Z3-N0/flexlog/server"
	"github.com/Z3-N0/flexlog/web"
	"golang.org/x/sync/errgroup"
)

const (
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 15 * time.Second
	defaultReadHeaderTimeout = 10 * time.Second
	defaultIdleTimeout       = 30 * time.Second
	defaultShutdownTimeout   = 10 * time.Second
)

func start(params *Params, logger *flexlog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	info, err := os.Stat(params.Path)
	if err != nil {
		return fmt.Errorf("cannot stat path: %w", err)
	}

	var scan server.ScanResult
	logger.Debug(ctx, "scanning path", "path", params.Path)
	if info.IsDir() {
		scan, err = server.ScanDir(ctx, logger, params.Path)
	} else {
		scan, err = server.ScanFile(params.Path)
	}
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}
	logger.Info(ctx, "scan complete", "files_found", len(scan.Files))

	handler, err := web.Initialize(ctx, scan, logger, params.PageSize, params.Port)
	if err != nil {
		return fmt.Errorf("web init failed: %w", err)
	}

	addr := fmt.Sprintf(":%d", params.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}

	grp, grpCtx := errgroup.WithContext(ctx)

	grp.Go(func() error {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	})

	grp.Go(func() error {
		<-grpCtx.Done()
		logger.Info(grpCtx, "shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
		logger.Info(grpCtx, "server shut down cleanly")
		return nil
	})

	return grp.Wait()
}
