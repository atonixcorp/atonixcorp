package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type EnvConfig struct {
	Environment struct {
		Name   string `yaml:"name"`
		Region string `yaml:"region"`
		NodeID string `yaml:"node_id"`
	} `yaml:"environment"`

	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Name     string `yaml:"name"`
		SSLMode  string `yaml:"sslmode"`
	} `yaml:"database"`

	Reconciler struct {
		IntervalSeconds int `yaml:"interval_seconds"`
	} `yaml:"reconciler"`

	Agent struct {
		Enabled             bool   `yaml:"enabled"`
		SyncIntervalSeconds int    `yaml:"sync_interval_seconds"`
		OVSBridge           string `yaml:"ovs_bridge"`
		DHCPConfigDir       string `yaml:"dhcp_config_dir"`
	} `yaml:"agent"`
}

func LoadEnv(path string) (*EnvConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg EnvConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *EnvConfig) DatabaseConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.SSLMode,
	)
}
