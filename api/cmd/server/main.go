package main

import (
	"context"
	"log"
	"os"
	"strings"

	"atonixcorp/api/internal/config"
	"atonixcorp/api/internal/containers"
	"atonixcorp/api/internal/controllers"
	"atonixcorp/api/internal/database"
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
		storageDriver, err = storage.NewCinderDriver(storage.CinderConfig{
			AuthURL: os.Getenv("CINDER_AUTH_URL"), Username: os.Getenv("CINDER_USERNAME"),
			Password: os.Getenv("CINDER_PASSWORD"), ProjectID: os.Getenv("CINDER_PROJECT_ID"),
			UserDomainID: os.Getenv("CINDER_USER_DOMAIN_ID"), Region: os.Getenv("CINDER_REGION"),
			Interface:     os.Getenv("CINDER_INTERFACE"),
			VolumeTypeIDs: splitCSV(os.Getenv("CINDER_VOLUME_TYPE_IDS")),
			Zones:         splitCSV(os.Getenv("CINDER_AVAILABILITY_ZONES")),
		})
		if err != nil {
			log.Fatalf("configure Cinder storage driver: %v", err)
		}
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
	if storageDriver != nil {
		go storageService.RunWorker(context.Background())
	} else {
		log.Print("storage provisioning disabled until a storage driver is configured")
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
	routes.SetupRoutes(r, authController, contactController, blogController, containerService, storageService)

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
