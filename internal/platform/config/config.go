package config

import "server/internal/platform/redisdb"

// Config contains process defaults for local demo servers.
type Config struct {
	HTTPAddr        string
	StateGRPCAddr   string
	Redis           redisdb.Config
	RCenterGRPCAddr string
}

// Default returns the local development configuration.
func Default() Config {
	return Config{
		HTTPAddr:        ":8080",
		StateGRPCAddr:   "127.0.0.1:9001",
		Redis:           redisdb.DefaultConfig(),
		RCenterGRPCAddr: "127.0.0.1:9002",
	}
}
