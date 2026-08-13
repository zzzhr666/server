package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"server/internal/platform/config"
	"server/internal/platform/logging"
	"syscall"
)

const serviceName = "state-server"

func main() {
	cfg := config.Default()
	if err := configureLogging(cfg); err != nil {
		log.Fatalf("configure logger failed: %v", err)
	}
	defer closeLogging()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runStateServer(ctx, cfg); err != nil {
		logging.Fatal("state-server stopped: %v", err)
	}
}

func configureLogging(cfg config.Config) error {
	level, err := logging.ParseLevel(cfg.LogConfig.LogLevel)
	if err != nil {
		return err
	}
	mode, err := logging.ParseMode(cfg.LogConfig.LogMode)
	if err != nil {
		return err
	}
	return logging.ConfigureDefault(logging.DefaultLoggerOptions{
		Level:       level,
		Mode:        mode,
		ServiceName: serviceName,
		LogDir:      "./logs",
	})
}

func closeLogging() {
	if err := logging.CloseDefault(); err != nil {
		log.Printf("close logger failed: %v", err)
	}
}

func runStateServer(ctx context.Context, cfg config.Config) error {
	app, err := newApplication(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := app.Close(closeCtx); err != nil {
			logging.Error("close application resources failed: %v", err)
		}
	}()
	return app.Run(ctx)
}
