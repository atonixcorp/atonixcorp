package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type HPCConfig struct {
	Service struct {
		Name string `yaml:"name"`
		Port int    `yaml:"port"`
	} `yaml:"service"`

	Auth struct {
		JWTSigningKey string `yaml:"jwt_signing_key"`
	} `yaml:"auth"`

	Scheduler struct {
		ExecutionProfiles map[string]ExecutionProfile `yaml:"execution_profiles"`
		DefaultBackend    string                      `yaml:"default_backend"`
		Slurm             struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"slurm"`
		K8s struct {
			Enabled   bool   `yaml:"enabled"`
			Namespace string `yaml:"namespace"`
		} `yaml:"k8s"`
	} `yaml:"scheduler"`

	Storage struct {
		S3 struct {
			Endpoint  string `yaml:"endpoint"`
			Bucket    string `yaml:"bucket"`
			AccessKey string `yaml:"access_key"`
			SecretKey string `yaml:"secret_key"`
			Secure    bool   `yaml:"secure"`
		} `yaml:"s3"`
	} `yaml:"storage"`

	Database struct {
		Postgres struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			User     string `yaml:"user"`
			Password string `yaml:"password"`
			DBName   string `yaml:"dbname"`
		} `yaml:"postgres"`
	} `yaml:"database"`

	Metrics struct {
		Port int `yaml:"port"`
	} `yaml:"metrics"`

	Cluster struct {
		DiscoveryInterval string `yaml:"discovery_interval"`
		HealthInterval    string `yaml:"health_interval"`
	} `yaml:"cluster"`
}

type ExecutionProfile struct {
	Backend   string `yaml:"backend"`
	Partition string `yaml:"partition"`
	Account   string `yaml:"account"`
}

func Load(path string) (*HPCConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg HPCConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validate(cfg *HPCConfig) error {
	if cfg.Service.Port == 0 {
		return fmt.Errorf("service.port is required")
	}
	if cfg.Auth.JWTSigningKey == "" {
		return fmt.Errorf("auth.jwt_signing_key is required")
	}
	if cfg.Scheduler.DefaultBackend == "" {
		return fmt.Errorf("scheduler.default_backend is required")
	}
	if cfg.Storage.S3.Endpoint == "" {
		return fmt.Errorf("storage.s3.endpoint is required")
	}
	if _, err := time.ParseDuration(cfg.Cluster.DiscoveryInterval); err != nil {
		return fmt.Errorf("invalid cluster.discovery_interval: %v", err)
	}
	if _, err := time.ParseDuration(cfg.Cluster.HealthInterval); err != nil {
		return fmt.Errorf("invalid cluster.health_interval: %v", err)
	}
	return nil
}
