package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	databaseconfig "atonixcorp/cloud/database/config"
	"atonixcorp/cloud/database/core"
	"atonixcorp/cloud/database/openstack"
	"atonixcorp/cloud/database/services"
	"atonixcorp/cloud/database/store"

	"gopkg.in/yaml.v3"
)

type runtimeConfig struct {
	DefaultRegion string `yaml:"default_region"`
}

type plansConfig struct {
	Plans map[string]core.Plan `yaml:"plans"`
}

func main() {
	configDir := envOrDefault("DATABASE_CONFIG_DIR", "./config")
	var env runtimeConfig
	var plans plansConfig
	if err := loadYAML(configDir+"/env.yaml", &env); err != nil {
		log.Fatalf("load database environment config: %v", err)
	}
	if err := loadYAML(configDir+"/plans.yaml", &plans); err != nil {
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

	engineImages := map[string]string{
		"postgres:16":  os.Getenv("OPENSTACK_DATABASE_IMAGE_POSTGRES_16"),
		"postgres:17":  os.Getenv("OPENSTACK_DATABASE_IMAGE_POSTGRES_17"),
		"mysql:8.4":    os.Getenv("OPENSTACK_DATABASE_IMAGE_MYSQL_8_4"),
		"mariadb:11.4": os.Getenv("OPENSTACK_DATABASE_IMAGE_MARIADB_11_4"),
		"redis:7":      os.Getenv("OPENSTACK_DATABASE_IMAGE_REDIS_7"),
	}
	osConfig := openstack.AuthConfig{
		AuthURL:                     os.Getenv("OPENSTACK_AUTH_URL"),
		ApplicationCredentialID:     os.Getenv("OPENSTACK_APPLICATION_CREDENTIAL_ID"),
		ApplicationCredentialSecret: os.Getenv("OPENSTACK_APPLICATION_CREDENTIAL_SECRET"),
		Username:                    os.Getenv("OPENSTACK_USERNAME"),
		Password:                    os.Getenv("OPENSTACK_PASSWORD"),
		Domain:                      envOrDefault("OPENSTACK_USER_DOMAIN", "default"),
		ProjectID:                   firstConfigured(os.Getenv("OPENSTACK_SHARED_PROJECT_ID"), os.Getenv("OPENSTACK_PROJECT_ID")),
		Region:                      envOrDefault("OPENSTACK_REGION", env.DefaultRegion),
		Interface:                   envOrDefault("OPENSTACK_INTERFACE", "public"),
		AllowInsecureHTTP: strings.EqualFold(
			strings.TrimSpace(os.Getenv("OPENSTACK_ALLOW_INSECURE_HTTP")), "true",
		),
		ComputeEndpoint: os.Getenv("OPENSTACK_COMPUTE_ENDPOINT"),
		VolumeEndpoint:  os.Getenv("OPENSTACK_VOLUME_ENDPOINT"),
		NetworkEndpoint: os.Getenv("OPENSTACK_NETWORK_ENDPOINT"),
		DatabaseImages:  engineImages,
		DatabaseNetwork: os.Getenv("OPENSTACK_DATABASE_NETWORK_ID"),
		SecurityGroupID: os.Getenv("OPENSTACK_DATABASE_SECURITY_GROUP_ID"),
	}
	client, err := openstack.NewClient(osConfig)
	if err != nil {
		log.Fatalf("configure OpenStack database provider: %v", err)
	}
	flavors := make(map[string]string, len(plans.Plans))
	for id := range plans.Plans {
		flavors[id] = strings.TrimSpace(os.Getenv("OPENSTACK_DATABASE_FLAVOR_" + strings.ToUpper(id)))
	}
	provider := openstack.NewDatabaseProvisioner(openstack.NewServiceClient(client), openstack.ProvisionerConfig{
		EngineImages: engineImages, PlanFlavors: flavors, NetworkID: osConfig.DatabaseNetwork, SecurityGroupID: osConfig.SecurityGroupID, Region: osConfig.Region,
	})
	controller := core.NewController(metadataStore, provider)
	controller.SetPlans(plans.Plans)
	controller.SetDefaultRegion(envOrDefault("DATABASE_DEFAULT_REGION", env.DefaultRegion))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	services.NewProvisioningService(controller).RunWorker(ctx)
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

func firstConfigured(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
