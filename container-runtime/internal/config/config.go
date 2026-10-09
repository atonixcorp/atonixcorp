package config

import (
	"os"
)

type Config struct {
	RegistryURL string
	NodeName    string
	Env         string
}

func Load() *Config {
	return &Config{
		RegistryURL: getEnv("REGISTRY_URL", "registry.local"),
		NodeName:    getEnv("NODE_NAME", "node-1"),
		Env:         getEnv("ENV", "dev"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
