package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"server/internal/platform/config"
	"server/internal/platform/logging"
	"strings"
	"syscall"
)

type serverOptions struct {
	config     config.Config
	serverName string
}

func main() {
	options := loadOptions()
	if err := configureLogging(options); err != nil {
		log.Fatalf("configure logger failed: %v", err)
	}
	defer closeLogging()

	if err := runLogicServer(options); err != nil {
		logging.Fatal("logic-server stopped: %v", err)
	}
}

func loadOptions() serverOptions {
	cfg := config.Default()
	port := flag.String("port", "", "HTTP listen port")
	shortPort := flag.String("p", "", "HTTP listen port")
	serverName := flag.String("name", "logic-default", "logic server instance name")
	flag.Parse()

	if addr := listenAddrFromPort(*port); addr != "" {
		cfg.HTTPAddr = addr
	}
	if addr := listenAddrFromPort(*shortPort); addr != "" {
		cfg.HTTPAddr = addr
	}
	return serverOptions{
		config:     cfg,
		serverName: *serverName,
	}
}

func configureLogging(options serverOptions) error {
	level, err := logging.ParseLevel(options.config.LogConfig.LogLevel)
	if err != nil {
		return err
	}
	mode, err := logging.ParseMode(options.config.LogConfig.LogMode)
	if err != nil {
		return err
	}
	return logging.ConfigureDefault(logging.DefaultLoggerOptions{
		Level:       level,
		Mode:        mode,
		ServiceName: options.serverName,
		LogDir:      "./logs",
	})
}

func closeLogging() {
	if err := logging.CloseDefault(); err != nil {
		log.Printf("close logger failed: %v", err)
	}
}

func runLogicServer(options serverOptions) error {
	app, err := newApplication(options.config, options.serverName)
	if err != nil {
		return err
	}
	defer func() {
		if err := app.Close(); err != nil {
			logging.Error("close application resources failed: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return app.Run(ctx)
}

func listenAddrFromPort(port string) string {
	if port == "" {
		return ""
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}
