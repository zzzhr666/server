package mongodb

import "go.mongodb.org/mongo-driver/v2/mongo/options"

type Config struct {
	URI      string
	Database string
}

func DefaultConfig() Config {
	return Config{
		URI:      "mongodb://localhost:27017",
		Database: "game",
	}
}

func ClientOptions(config Config) *options.ClientOptions {
	return options.Client().ApplyURI(config.URI)
}
