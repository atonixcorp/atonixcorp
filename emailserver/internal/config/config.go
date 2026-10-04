package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SMTP   SMTPConfig   `yaml:"smtp"`
	IMAP   IMAPConfig   `yaml:"imap"`
	POP3   POP3Config   `yaml:"pop3"`
	API    APIConfig    `yaml:"api"`
	Queue  QueueConfig  `yaml:"queue"`
	Blob   BlobConfig   `yaml:"blob"`
	Spam   SpamConfig   `yaml:"spam"`
	Domain DomainConfig `yaml:"domain"`
	DB     DBConfig     `yaml:"database"`
	Log    LogConfig    `yaml:"log"`
}

type SMTPConfig struct {
	ListenAddr string `yaml:"listen_addr"`
	Hostname   string `yaml:"hostname"`
	TLSCert    string `yaml:"tls_cert"`
	TLSKey     string `yaml:"tls_key"`
}

type IMAPConfig struct {
	ListenAddr string `yaml:"listen_addr"`
}

type POP3Config struct {
	ListenAddr string `yaml:"listen_addr"`
}

type APIConfig struct {
	ListenAddr string `yaml:"listen_addr"`
}

type QueueConfig struct {
	WorkerCount int `yaml:"worker_count"`
}

type BlobConfig struct {
	Backend  string `yaml:"backend"` // s3, minio, local, ceph, azure, gcs
	Bucket   string `yaml:"bucket"`
	BasePath string `yaml:"base_path"` // for local backend
	Endpoint string `yaml:"endpoint"`  // for s3/minio
	Region   string `yaml:"region"`
}

type SpamConfig struct {
	Enable      bool   `yaml:"enable"`
	Blocklist   string `yaml:"blocklist"`
	MaxBodySize int    `yaml:"max_body_size"`
	RateLimit   int    `yaml:"rate_limit"`
	RateWindow  int    `yaml:"rate_window"`
}

type DomainConfig struct {
	DefaultDomain string `yaml:"default_domain"`
}

type DBConfig struct {
	DSN string `yaml:"dsn"`
}

type LogConfig struct {
	Level string `yaml:"level"` // info, debug, error
}

// ─────────────────────────────────────────────
// Load YAML config file
// ─────────────────────────────────────────────

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// ─────────────────────────────────────────────
// Environment overrides (optional)
// ─────────────────────────────────────────────

func (c *Config) ApplyEnv() {
	if v := os.Getenv("EMAILSERVER_LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
	if v := os.Getenv("EMAILSERVER_DB_DSN"); v != "" {
		c.DB.DSN = v
	}
	if v := os.Getenv("EMAILSERVER_SMTP_ADDR"); v != "" {
		c.SMTP.ListenAddr = v
	}
}
