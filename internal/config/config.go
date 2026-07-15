package config

import "learning/internal/redisdb"

type Config struct {
	HTTPAddr string
	Redis    redisdb.Config
}

func Default() Config {
	return Config{
		HTTPAddr: ":8080",
		Redis:    redisdb.DefaultConfig(),
	}
}
