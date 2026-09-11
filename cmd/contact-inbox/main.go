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

	"github.com/infrai-examples/healthtech-contact-inbox/internal/contact"
	"github.com/infrai-examples/healthtech-contact-inbox/internal/infrai"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	client, err := infrai.NewClient(os.Getenv("INFRAI_API_KEY"), nil)
	if err != nil {
		logger.Error("configuration rejected", "error", err)
		os.Exit(1)
	}
	handler, err := contact.NewHandler(client, os.Getenv("TEAM_INBOX"), logger)
	if err != nil {
		logger.Error("configuration rejected", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.Handle("POST /contact", handler)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	server := &http.Server{
		Addr:              envOr("LISTEN_ADDR", ":8080"),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		logger.Info("contact inbox listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server stopped", "error", err)
			os.Exit(1)
		}
	}()

	<-stopped
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
