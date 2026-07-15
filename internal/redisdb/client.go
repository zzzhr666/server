package redisdb

import (
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr     string
	Password string
	DB       int
}

func DefaultConfig() Config {
	return Config{
		Addr: "127.0.0.1:6379",
		DB:   0,
	}
}

func NewClient(config Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})
}
