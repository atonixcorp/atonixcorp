package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type SlurmConfig struct {
	Enabled        bool   `yaml:"enabled"`
	SbatchPath     string `yaml:"sbatch_path"`
	SqueuePath     string `yaml:"squeue_path"`
	SacctPath      string `yaml:"sacct_path"`
	ScancelPath    string `yaml:"scancel_path"`
	ScontrolPath   string `yaml:"scontrol_path"`
	CommandTimeout string `yaml:"command_timeout"`
}

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
		Slurm             SlurmConfig                 `yaml:"slurm"`
		K8s               struct {
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
			SSLMode  string `yaml:"sslmode"`
		} `yaml:"postgres"`
	} `yaml:"database"`

	Metrics struct {
		Port int `yaml:"port"`
	} `yaml:"metrics"`

	Cluster struct {
		DiscoveryInterval string `yaml:"discovery_interval"`
		HealthInterval    string `yaml:"health_interval"`
		HealthPort        int    `yaml:"health_port"`
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

	if err := applyEnvironment(&cfg); err != nil {
		return nil, err
	}
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyEnvironment(cfg *HPCConfig) error {
	stringValues := map[string]*string{
		"HPC_SERVICE_NAME":               &cfg.Service.Name,
		"HPC_JWT_SIGNING_KEY":            &cfg.Auth.JWTSigningKey,
		"HPC_DEFAULT_BACKEND":            &cfg.Scheduler.DefaultBackend,
		"HPC_SLURM_SBATCH_PATH":          &cfg.Scheduler.Slurm.SbatchPath,
		"HPC_SLURM_SQUEUE_PATH":          &cfg.Scheduler.Slurm.SqueuePath,
		"HPC_SLURM_SACCT_PATH":           &cfg.Scheduler.Slurm.SacctPath,
		"HPC_SLURM_SCANCEL_PATH":         &cfg.Scheduler.Slurm.ScancelPath,
		"HPC_SLURM_SCONTROL_PATH":        &cfg.Scheduler.Slurm.ScontrolPath,
		"HPC_SLURM_COMMAND_TIMEOUT":      &cfg.Scheduler.Slurm.CommandTimeout,
		"HPC_K8S_NAMESPACE":              &cfg.Scheduler.K8s.Namespace,
		"HPC_S3_ENDPOINT":                &cfg.Storage.S3.Endpoint,
		"HPC_S3_BUCKET":                  &cfg.Storage.S3.Bucket,
		"HPC_S3_ACCESS_KEY":              &cfg.Storage.S3.AccessKey,
		"HPC_S3_SECRET_KEY":              &cfg.Storage.S3.SecretKey,
		"HPC_DB_HOST":                    &cfg.Database.Postgres.Host,
		"HPC_DB_USER":                    &cfg.Database.Postgres.User,
		"HPC_DB_PASSWORD":                &cfg.Database.Postgres.Password,
		"HPC_DB_NAME":                    &cfg.Database.Postgres.DBName,
		"HPC_DB_SSLMODE":                 &cfg.Database.Postgres.SSLMode,
		"HPC_CLUSTER_DISCOVERY_INTERVAL": &cfg.Cluster.DiscoveryInterval,
		"HPC_CLUSTER_HEALTH_INTERVAL":    &cfg.Cluster.HealthInterval,
	}
	for name, target := range stringValues {
		if value, ok := os.LookupEnv(name); ok {
			*target = value
		}
	}

	booleanValues := map[string]*bool{
		"HPC_SLURM_ENABLED": &cfg.Scheduler.Slurm.Enabled,
		"HPC_K8S_ENABLED":   &cfg.Scheduler.K8s.Enabled,
		"HPC_S3_SECURE":     &cfg.Storage.S3.Secure,
	}
	for name, target := range booleanValues {
		if value, ok := os.LookupEnv(name); ok {
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("%s must be true or false: %w", name, err)
			}
			*target = parsed
		}
	}

	integerValues := map[string]*int{
		"HPC_SERVICE_PORT":        &cfg.Service.Port,
		"HPC_METRICS_PORT":        &cfg.Metrics.Port,
		"HPC_DB_PORT":             &cfg.Database.Postgres.Port,
		"HPC_CLUSTER_HEALTH_PORT": &cfg.Cluster.HealthPort,
	}
	for name, target := range integerValues {
		if value, ok := os.LookupEnv(name); ok {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("%s must be an integer: %w", name, err)
			}
			*target = parsed
		}
	}
	return nil
}

func validate(cfg *HPCConfig) error {
	if cfg.Service.Port < 1 || cfg.Service.Port > 65535 {
		return fmt.Errorf("service.port must be between 1 and 65535")
	}
	if cfg.Metrics.Port < 1 || cfg.Metrics.Port > 65535 {
		return fmt.Errorf("metrics.port must be between 1 and 65535")
	}
	if len(cfg.Auth.JWTSigningKey) < 32 {
		return fmt.Errorf("HPC_JWT_SIGNING_KEY must contain at least 32 characters")
	}
	switch cfg.Scheduler.DefaultBackend {
	case "slurm", "k8s", "none":
	default:
		return fmt.Errorf("scheduler.default_backend must be slurm, k8s, or none")
	}

	if strings.TrimSpace(cfg.Storage.S3.Endpoint) == "" ||
		strings.TrimSpace(cfg.Storage.S3.Bucket) == "" ||
		cfg.Storage.S3.AccessKey == "" || cfg.Storage.S3.SecretKey == "" {
		return fmt.Errorf("HPC_S3_ENDPOINT, HPC_S3_BUCKET, HPC_S3_ACCESS_KEY, and HPC_S3_SECRET_KEY are required")
	}
	if cfg.Database.Postgres.Host == "" || cfg.Database.Postgres.Port < 1 ||
		cfg.Database.Postgres.User == "" || cfg.Database.Postgres.Password == "" ||
		cfg.Database.Postgres.DBName == "" {
		return fmt.Errorf("HPC database host, port, user, password, and name are required")
	}
	if cfg.Database.Postgres.SSLMode == "" {
		return fmt.Errorf("HPC_DB_SSLMODE is required")
	}
	for name, value := range map[string]string{
		"cluster.discovery_interval":      cfg.Cluster.DiscoveryInterval,
		"cluster.health_interval":         cfg.Cluster.HealthInterval,
		"scheduler.slurm.command_timeout": cfg.Scheduler.Slurm.CommandTimeout,
	} {
		duration, err := time.ParseDuration(value)
		if err != nil || duration <= 0 {
			return fmt.Errorf("%s must be a positive duration", name)
		}
	}
	if cfg.Cluster.HealthPort < 1 || cfg.Cluster.HealthPort > 65535 {
		return fmt.Errorf("cluster.health_port must be between 1 and 65535")
	}
	if cfg.Scheduler.K8s.Enabled && cfg.Scheduler.K8s.Namespace == "" {
		return fmt.Errorf("HPC_K8S_NAMESPACE is required when Kubernetes is enabled")
	}
	if cfg.Scheduler.Slurm.Enabled {
		for name, value := range map[string]string{
			"HPC_SLURM_SBATCH_PATH":   cfg.Scheduler.Slurm.SbatchPath,
			"HPC_SLURM_SQUEUE_PATH":   cfg.Scheduler.Slurm.SqueuePath,
			"HPC_SLURM_SACCT_PATH":    cfg.Scheduler.Slurm.SacctPath,
			"HPC_SLURM_SCANCEL_PATH":  cfg.Scheduler.Slurm.ScancelPath,
			"HPC_SLURM_SCONTROL_PATH": cfg.Scheduler.Slurm.ScontrolPath,
		} {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s cannot be empty when Slurm is enabled", name)
			}
		}
	}
	return nil
}

func (cfg *HPCConfig) DatabaseDSN() string {
	postgres := cfg.Database.Postgres
	address := net.JoinHostPort(postgres.Host, strconv.Itoa(postgres.Port))
	databaseURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(postgres.User, postgres.Password),
		Host:   address,
		Path:   "/" + postgres.DBName,
	}
	query := url.Values{}
	query.Set("sslmode", postgres.SSLMode)
	databaseURL.RawQuery = query.Encode()
	return databaseURL.String()
}
