package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/code19m/sentinel/app"
	"github.com/code19m/sentinel/config"
)

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.ErrorContext(ctx, "Failed to start service", slog.Any("error", err))
		os.Exit(1)
	}

	app := app.New(logger, cfg)
	app.Start()
}
