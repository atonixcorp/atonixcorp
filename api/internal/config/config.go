package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBPort        string
	JWTSecret     string
	SMTPHost      string
	SMTPPort      string
	SMTPUser      string
	SMTPPassword  string
	SMTPRecipient string
}

// LoadConfig loads environment variables from .env and OS environment
func LoadConfig() *Config {
	// Load .env file if present
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, using system environment variables")
	}

	cfg := &Config{
		DBHost:        getEnv("DB_HOST"),
		DBUser:        getEnv("DB_USER"),
		DBPassword:    getEnv("DB_PASSWORD"),
		DBName:        getEnv("DB_NAME"),
		DBPort:        getEnv("DB_PORT"),
		JWTSecret:     getEnv("JWT_SECRET"),
		SMTPHost:      getEnv("SMTP_HOST"),
		SMTPPort:      getEnv("SMTP_PORT"),
		SMTPUser:      getEnv("SMTP_USER"),
		SMTPPassword:  getEnv("SMTP_PASSWORD"),
		SMTPRecipient: getEnv("SMTP_RECIPIENT"),
	}

	return cfg
}

// getEnv ensures required variables exist
func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("❌ Missing required environment variable: %s", key)
	}
	return value
}
