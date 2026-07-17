package config

import "server/internal/platform/redisdb"

type Config struct {
	HTTPAddr     string
	StateRPCAddr string
	Redis        redisdb.Config
}

func Default() Config {
	return Config{
		HTTPAddr:     ":8080",
		StateRPCAddr: "127.0.0.1:9001",
		Redis:        redisdb.DefaultConfig(),
	}
}
