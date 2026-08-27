package mongodb

import "go.mongodb.org/mongo-driver/v2/mongo/options"

type Config struct {
	URI      string
	Database string
}

// DefaultConfig 返回本地开发环境的 MongoDB 连接配置。
func DefaultConfig() Config {
	return Config{
		URI:      "mongodb://localhost:27017",
		Database: "game",
	}
}

// ClientOptions 将应用配置转换为 MongoDB 客户端选项。
func ClientOptions(config Config) *options.ClientOptions {
	return options.Client().ApplyURI(config.URI)
}
