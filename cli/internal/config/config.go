package config

import (
	"os"
)

type Config struct {
	APIBaseURL string
	JWTSecret  string
}

func Load() *Config {
	return &Config{
		APIBaseURL: os.Getenv("ATONIX_API"),
		JWTSecret:  os.Getenv("ATONIX_JWT"),
	}
}
