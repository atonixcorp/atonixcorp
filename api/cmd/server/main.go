package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"atonixcorp/api/internal/cloud"
	"atonixcorp/api/internal/config"
	"atonixcorp/api/internal/containers"
	"atonixcorp/api/internal/controllers"
	"atonixcorp/api/internal/database"
	"atonixcorp/api/internal/files"
	"atonixcorp/api/internal/objectstorage"
	minioadapter "atonixcorp/api/internal/objectstorage/minio"
	"atonixcorp/api/internal/repositories"
	"atonixcorp/api/internal/routes"
	"atonixcorp/api/internal/services"
	"atonixcorp/api/internal/storage"

	"github.com/gin-gonic/gin"
)

func main() {
	appConfig := config.LoadConfig()
	r := gin.Default()
	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"status": "ok"})
	})

	// Connect to PostgreSQL
	db := database.Connect()

	var objectProvider objectstorage.Provider
	minioProvider, objectErr := minioadapter.FromEnvironment()
	if objectErr != nil {
		log.Printf("object storage disabled: %v", objectErr)
	} else if minioProvider != nil {
		objectProvider = minioProvider
	} else {
		log.Print("object storage disabled until STORAGE_PROVIDER is configured")
	}
	fileService := files.NewService(db, objectProvider)
	go fileService.RunCleanup(context.Background())

	provisioner, err := containers.NewKubernetesProvisioner()
	if err != nil {
		log.Printf("container Kubernetes runtime disabled: %v", err)
	}
	secretKey, err := containers.SecretKeyFromEnvironment(os.Getenv("CONTAINER_SECRET_KEY"))
	if err != nil {
		log.Fatalf("configure container secret encryption: %v", err)
	}
	containerService := containers.NewService(db, provisioner, secretKey)
	if provisioner != nil {
		go containerService.RunWorker(context.Background())
		go containerService.RunReconciler(context.Background())
	} else {
		log.Print("container deployment API is read-only until Kubernetes credentials are configured")
	}
	var storageDriver storage.BlockDriver
	switch strings.ToLower(strings.TrimSpace(os.Getenv("STORAGE_DRIVER"))) {
	case "local-file":
		storageDriver, err = storage.NewLocalFileDriver(os.Getenv("STORAGE_DATA_DIR"))
		if err != nil {
			log.Fatalf("configure local storage driver: %v", err)
		}
	case "cinder":
	case "libvirt-qemu":
		storageDriver, err = storage.NewLibvirtQEMUDriver(storage.LibvirtQEMUConfig{
			URI: os.Getenv("LIBVIRT_URI"), StoragePath: os.Getenv("LIBVIRT_STORAGE_PATH"),
		})
		if err != nil {
			log.Fatalf("configure Libvirt/QEMU storage driver: %v", err)
		}
	case "":
	default:
		log.Fatalf("unsupported STORAGE_DRIVER %q", os.Getenv("STORAGE_DRIVER"))
	}
	storageService := storage.NewService(db, storageDriver)

	var openStackProvider cloud.Provider
	var openStackFactory cloud.ComputeProviderFactory
	openStackConfig, openStackConfigured, err := cloud.OpenStackConfigFromEnv()
	if err != nil {
		log.Fatalf("configure Nova integration: %v", err)
	}
	if openStackConfigured {
		authCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		provider, providerErr := cloud.NewOpenStackProvider(authCtx, openStackConfig)
		cancel()
		if providerErr != nil {
			log.Fatalf("initialize Nova integration: %v", providerErr)
		}
		openStackProvider = provider
		openStackFactory = provider
		log.Printf(
			"OpenStack provider factory configured for region %s; compute operations require a mapped organization account",
			openStackConfig.Region,
		)
	} else {
		log.Print("OpenStack integration disabled until OPENSTACK_AUTH_URL is configured")
	}
	accountEncryptionKey, err := cloud.AccountEncryptionKeyFromEnvironment(
		os.Getenv("OPENSTACK_ACCOUNT_ENCRYPTION_KEY"),
	)
	if err != nil {
		log.Fatalf("configure OpenStack account encryption: %v", err)
	}
	providerRegistry := cloud.NewComputeProviderRegistry()
	if openStackFactory != nil {
		if err := providerRegistry.Register(openStackFactory); err != nil {
			log.Fatalf("register OpenStack provider adapter: %v", err)
		}
	}
	cloudService := cloud.NewServiceWithRegistry(
		db, openStackProvider, providerRegistry, accountEncryptionKey,
	)
	storageRegistry := storage.NewBlockDriverRegistry()
	localDriverName := ""
	if storageDriver != nil {
		if err := storageRegistry.Register(storageDriver); err != nil {
			log.Fatalf("register storage driver: %v", err)
		}
		localDriverName = storageDriver.Name()
	}
	cinderConfig := storage.CinderConfig{
		AuthURL:                  firstNonEmpty(os.Getenv("CINDER_AUTH_URL"), os.Getenv("OPENSTACK_AUTH_URL")),
		Region:                   firstNonEmpty(os.Getenv("CINDER_REGION"), os.Getenv("OPENSTACK_REGION"), "RegionOne"),
		Interface:                firstNonEmpty(os.Getenv("CINDER_INTERFACE"), os.Getenv("OPENSTACK_INTERFACE"), "public"),
		VolumeTypeIDs:            splitCSV(os.Getenv("CINDER_VOLUME_TYPE_IDS")),
		EncryptedVolumeTypeIDs:   splitCSV(os.Getenv("CINDER_ENCRYPTED_VOLUME_TYPE_IDS")),
		UnencryptedVolumeTypeIDs: splitCSV(os.Getenv("CINDER_UNENCRYPTED_VOLUME_TYPE_IDS")),
		Zones:                    splitCSV(os.Getenv("CINDER_AVAILABILITY_ZONES")),
		AllowInsecureHTTP:        strings.EqualFold(strings.TrimSpace(os.Getenv("OPENSTACK_ALLOW_INSECURE_HTTP")), "true"),
	}
	if err := storageRegistry.RegisterFactory(storage.NewCinderDriverFactory(cinderConfig)); err != nil {
		log.Fatalf("register Cinder storage adapter: %v", err)
	}
	scopedStorageDrivers, err := storage.NewScopedDriversWithRegistry(
		cloudService, storageRegistry, localDriverName,
	)
	if err != nil {
		log.Fatalf("configure scoped storage drivers: %v", err)
	}
	storageService.SetScopedDrivers(scopedStorageDrivers)
	go storageService.RunWorker(context.Background())
	if openStackProvider != nil {
		go cloudService.RunReconciler(context.Background())
	}

	// Repositories
	userRepo := repositories.NewUserRepository(db)
	contactRepo := repositories.NewContactRepository(db)
	blogRepo := repositories.NewBlogRepository(db)

	// Services
	authService := services.NewAuthService(userRepo)
	contactService := services.NewContactService(contactRepo, appConfig)
	blogService := services.NewBlogService(blogRepo)

	// Controllers
	authController := controllers.NewAuthController(authService)
	contactController := controllers.NewContactController(contactService)
	blogController := controllers.NewBlogController(blogService)

	// Routes
	routes.SetupRoutes(
		r, authController, contactController, blogController, containerService, storageService,
		cloudService, fileService,
	)

	r.Run(":8080")
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
