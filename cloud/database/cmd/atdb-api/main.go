package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"atonixcorp/cloud/database/api"
	databaseconfig "atonixcorp/cloud/database/config"
	"atonixcorp/cloud/database/core"
	"atonixcorp/cloud/database/openstack"
	"atonixcorp/cloud/database/services"
	"atonixcorp/cloud/database/store"

	"gopkg.in/yaml.v3"
)

type EnvConfig struct {
	Environment   string `yaml:"environment"`
	DefaultRegion string `yaml:"default_region"`
}

type PlansConfig struct {
	Plans map[string]core.Plan `yaml:"plans"`
}

type engineImageSetting struct {
	engine  string
	version string
	envVar  string
}

var databaseEngineImages = []engineImageSetting{
	{"postgres", "16", "OPENSTACK_DATABASE_IMAGE_POSTGRES_16"},
	{"postgres", "17", "OPENSTACK_DATABASE_IMAGE_POSTGRES_17"},
	{"mysql", "8.4", "OPENSTACK_DATABASE_IMAGE_MYSQL_8_4"},
	{"mariadb", "11.4", "OPENSTACK_DATABASE_IMAGE_MARIADB_11_4"},
	{"redis", "7", "OPENSTACK_DATABASE_IMAGE_REDIS_7"},
}

func main() {
	configDir := strings.TrimSpace(os.Getenv("DATABASE_CONFIG_DIR"))
	if configDir == "" {
		configDir = "./config"
	}
	var envCfg EnvConfig
	var plansCfg PlansConfig
	if err := loadYAML(configDir+"/env.yaml", &envCfg); err != nil {
		log.Fatalf("load database environment config: %v", err)
	}
	if err := loadYAML(configDir+"/plans.yaml", &plansCfg); err != nil {
		log.Fatalf("load database plans: %v", err)
	}

	dsn, err := databaseconfig.DatabaseDSN()
	if err != nil {
		log.Fatal(err)
	}
	metadataStore, err := store.NewPostgresStore(dsn)
	if err != nil {
		log.Fatalf("connect database metadata store: %v", err)
	}
	defer metadataStore.Close()

	var osService core.OpenStackClient
	engineImages := make(map[string]string, len(databaseEngineImages))
	for _, setting := range databaseEngineImages {
		if value := strings.TrimSpace(os.Getenv(setting.envVar)); value != "" {
			engineImages[setting.engine+":"+setting.version] = value
		}
	}
	networkID := strings.TrimSpace(os.Getenv("OPENSTACK_DATABASE_NETWORK_ID"))
	securityGroupID := strings.TrimSpace(os.Getenv("OPENSTACK_DATABASE_SECURITY_GROUP_ID"))
	controller := core.NewController(metadataStore, nil)
	for id, plan := range plansCfg.Plans {
		plan.FlavorID = strings.TrimSpace(os.Getenv("OPENSTACK_DATABASE_FLAVOR_" + strings.ToUpper(id)))
		plansCfg.Plans[id] = plan
	}
	controller.SetPlans(plansCfg.Plans)

	openStackConfig := openstack.AuthConfig{
		AuthURL:         os.Getenv("OPENSTACK_AUTH_URL"),
		Username:        os.Getenv("OPENSTACK_USERNAME"),
		Password:        os.Getenv("OPENSTACK_PASSWORD"),
		Domain:          envOrDefault("OPENSTACK_USER_DOMAIN", "default"),
		ProjectID:       os.Getenv("OPENSTACK_PROJECT_ID"),
		ComputeEndpoint: os.Getenv("OPENSTACK_COMPUTE_ENDPOINT"),
		VolumeEndpoint:  os.Getenv("OPENSTACK_VOLUME_ENDPOINT"),
		NetworkEndpoint: os.Getenv("OPENSTACK_NETWORK_ENDPOINT"),
		DatabaseImages:  engineImages,
		DatabaseFlavors: configuredFlavors(plansCfg.Plans),
		DatabaseNetwork: networkID,
		SecurityGroupID: securityGroupID,
	}
	if hasOpenStackConfiguration(openStackConfig) {
		client, clientErr := openstack.NewClient(openStackConfig)
		if clientErr != nil {
			log.Printf("OpenStack provisioning disabled: %v", clientErr)
		} else {
			osService = openstack.NewServiceClient(client)
			controller = core.NewController(metadataStore, osService)
		}
	} else {
		log.Print("OpenStack provisioning is not configured; database API will run read-only")
	}
	controller.SetPlans(plansCfg.Plans)
	controller.SetProvisioningConfig(engineImages, networkID, securityGroupID)
	controller.SetDefaultRegion(envOrDefault("DATABASE_DEFAULT_REGION", envCfg.DefaultRegion))

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}
	apiServer := api.NewAPIServer(controller, metadataStore, jwtSecret)
	server := &http.Server{
		Addr:              envOrDefault("DATABASE_API_ADDR", ":8085"),
		Handler:           apiServer.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if osService != nil {
		provisioning := services.NewProvisioningService(controller)
		go provisioning.RunWorker(ctx)
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown database API: %v", err)
		}
	}()

	log.Printf("database API listening on %s (region %s; environment %s)",
		server.Addr, envCfg.DefaultRegion, envCfg.Environment)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("database API failed: %v", err)
	}
}

func loadYAML(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func hasOpenStackConfiguration(cfg openstack.AuthConfig) bool {
	return strings.TrimSpace(cfg.AuthURL) != "" &&
		strings.TrimSpace(cfg.Username) != "" &&
		strings.TrimSpace(cfg.Password) != "" &&
		strings.TrimSpace(cfg.ProjectID) != "" &&
		strings.TrimSpace(cfg.ComputeEndpoint) != "" &&
		strings.TrimSpace(cfg.VolumeEndpoint) != ""
}

func configuredFlavors(plans map[string]core.Plan) map[string]string {
	flavors := make(map[string]string, len(plans))
	for name, plan := range plans {
		if plan.FlavorID != "" {
			flavors[name] = plan.FlavorID
		}
	}
	return flavors
}
