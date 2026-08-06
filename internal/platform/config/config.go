package config

import "server/internal/platform/redisdb"

// Config 包含本地演示服务进程的默认配置。
type Config struct {
	HTTPAddr        string
	StateGRPCAddr   string
	Redis           redisdb.Config
	RCenterGRPCAddr string
}

// Default 返回本地开发环境配置。
func Default() Config {
	return Config{
		HTTPAddr:        ":8080",
		StateGRPCAddr:   "127.0.0.1:9001",
		Redis:           redisdb.DefaultConfig(),
		RCenterGRPCAddr: "127.0.0.1:9002",
	}
}
