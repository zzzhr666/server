package redisdb

import (
	"github.com/redis/go-redis/v9"
)

// Config 描述 Redis 客户端的连接参数。
type Config struct {
	// Addr 是 Redis 服务地址。
	Addr string
	// Password 是 Redis 认证密码。
	Password string
	// DB 是要使用的 Redis 逻辑数据库编号。
	DB int
}

// DefaultConfig 返回本地开发环境的 Redis 连接配置。
func DefaultConfig() Config {
	return Config{
		Addr: "127.0.0.1:6379",
		DB:   0,
	}
}

// NewClient 根据配置创建 Redis 客户端。
func NewClient(config Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})
}
