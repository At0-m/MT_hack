package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"tramflow/internal/auth"
	"tramflow/internal/config"
	"tramflow/internal/contract"
	"tramflow/internal/engine"
	"tramflow/internal/httpapi"
	"tramflow/internal/inference"
	"tramflow/internal/storage"
)

func run() error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	spec, err := contract.Load(c.Contract)
	if err != nil {
		return err
	}
	store, err := storage.Open(ctx, c.Database, c.Pool)
	if err != nil {
		return err
	}
	defer store.Pool.Close()
	models := inference.New(c.Artifacts)
	defer models.Close()
	svc := &engine.Service{Repo: store, Models: models, Cache: engine.NewCache(64 * 1024 * 1024)}
	api := &httpapi.Server{
		TrustedProxies: c.TrustedProxies,
		Store:          store,
		Models:         models,
		Service:        svc,
		Contract:       spec,
		Auth:           auth.New(store),
		CookieSecure:   os.Getenv("COOKIE_SECURE") != "false",
		Origin:         c.Origin,
	}
	server := &http.Server{
		Addr:              c.Addr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16384,
	}
	done := make(chan error, 1)
	go func() {
		slog.Info("api started", "address", c.Addr)
		done <- server.ListenAndServe()
	}()
	select {
	case err := <-done:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("api stopped", "error", err)
		os.Exit(1)
	}
}
