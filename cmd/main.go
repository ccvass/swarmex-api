package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	api "github.com/ccvass/swarmex/swarmex-api"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPath := os.Getenv("SWARMEX_DB_PATH")
	if dbPath == "" {
		dbPath = "/data/swarmex-api.db"
	}

	srv, err := api.New(dbPath, logger)
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer srv.Close()

	go func() { logger.Info("api server", "addr", ":8080"); http.ListenAndServe(":8080", srv.Handler()) }()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	logger.Info("swarmex-api starting")
	<-ctx.Done()
	logger.Info("shutdown complete")
}
