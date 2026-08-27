package config

import (
	"os"
	"server/internal/platform/mongodb"
	"server/internal/platform/redisdb"
)

// Config 包含本地演示服务进程的默认配置。
type Config struct {
	HTTPAddr        string
	StateGRPCAddr   string
	Redis           redisdb.Config
	RCenterGRPCAddr string
	TCPAddr         string
	Mongo           mongodb.Config
	LogConfig       LogConfig
	MetricsAddr     string
}

type LogConfig struct {
	LogLevel string
	LogMode  string
}

// Default 返回本地开发环境配置。
func Default() Config {
	redisConfig := redisdb.DefaultConfig()
	redisConfig.Addr = envOrDefault("REDIS_ADDR", redisConfig.Addr)
	mongoConfig := mongodb.DefaultConfig()
	mongoConfig.URI = envOrDefault("MONGO_URI", mongoConfig.URI)
	mongoConfig.Database = envOrDefault("MONGO_DATABASE", mongoConfig.Database)
	return Config{
		HTTPAddr:        envOrDefault("HTTP_ADDR", ":8080"),
		StateGRPCAddr:   envOrDefault("STATE_GRPC_ADDR", "127.0.0.1:9001"),
		Redis:           redisConfig,
		RCenterGRPCAddr: envOrDefault("RCENTER_GRPC_ADDR", "127.0.0.1:9002"),
		TCPAddr:         envOrDefault("TCP_ADDR", ":8081"),
		Mongo:           mongoConfig,
		LogConfig:       DefaultLogConfig(),
		MetricsAddr:     envOrDefault("METRICS_ADDR", ":9200"),
	}
}

// DefaultLogConfig 返回由环境变量覆盖的默认日志配置。
func DefaultLogConfig() LogConfig {
	return LogConfig{
		LogLevel: envOrDefault("LOG_LEVEL", "info"),
		LogMode:  envOrDefault("LOG_MODE", "debug"),
	}
}

func envOrDefault(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}
