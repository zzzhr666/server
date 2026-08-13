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

const serviceName = "rcenter-server"

func main() {
	cfg := config.Default()
	if err := configureLogging(cfg); err != nil {
		log.Fatalf("configure logger failed: %v", err)
	}
	defer closeLogging()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runRCenterServer(ctx, cfg); err != nil {
		logging.Fatal("rcenter-server stopped: %v", err)
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

func runRCenterServer(ctx context.Context, cfg config.Config) error {
	app, err := newApplication(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := app.Close(); err != nil {
			logging.Error("close application resources failed: %v", err)
		}
	}()
	return app.Run(ctx)
}
