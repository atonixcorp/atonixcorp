package routes

import (
	"atonixcorp/api/internal/cloud"
	"atonixcorp/api/internal/containers"
	"atonixcorp/api/internal/controllers"
	"atonixcorp/api/internal/database"
	"atonixcorp/api/internal/files"
	"atonixcorp/api/internal/security/middleware"
	"atonixcorp/api/internal/storage"

	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	r *gin.Engine,
	authController *controllers.AuthController,
	contactController *controllers.ContactController,
	blogController *controllers.BlogController,
	containerService *containers.Service,
	storageService *storage.Service,
	cloudService *cloud.Service,
	fileService *files.Service,
) {

	r.Use(middleware.SecurityHeaders())

	// -------------------------
	// API VERSIONING
	// -------------------------
	api := r.Group("/api/v1")

	// -------------------------
	// PUBLIC CONTACT ROUTES
	// -------------------------
	contact := api.Group("/contact")
	{
		contact.POST("/sponsorship", contactController.SubmitSponsorship)
	}

	auth := api.Group("/auth")
	{
		auth.POST("/signup", authController.Signup)
		auth.POST("/login", authController.Login)
	}

	// -------------------------
	// AUTHENTICATED API ROUTES
	// -------------------------
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(os.Getenv("JWT_SECRET"), database.DB))
	platformAdmin := protected.Group("/admin")
	platformAdmin.Use(middleware.RequirePlatformSuperAdmin())
	platformAdminController := controllers.NewPlatformAdminController(database.DB)
	platformAdmin.GET("/users", platformAdminController.ListUsers)
	platformAdmin.PATCH("/users/:userID/platform-role", platformAdminController.UpdateUserRole)

	contactMessages := protected.Group("/contact")
	{
		contactMessages.POST("/submit", contactController.Submit)
		contactMessages.GET("/messages", middleware.RequirePlatformSuperAdmin(), contactController.GetAll)
	}

	protected.GET("/auth/me", authController.Session)

	support := protected.Group("/support/tickets")
	support.Use(middleware.SecurityRequestMetadata(), middleware.SecurityRateLimit(30, time.Minute))
	{
		support.POST("", contactController.SubmitSupportTicket)
		support.GET("", contactController.ListSupportTickets)
	}

	observabilityController := controllers.NewObservabilityController(database.DB)
	observabilityRoutes := protected.Group("/observability/tenants/:tenantID/projects/:projectID")
	{
		observabilityRoutes.GET("/overview", observabilityController.Overview)
		observabilityRoutes.GET("/resources", observabilityController.Resources)
		observabilityRoutes.GET("/resources/:resourceType/:resourceID", observabilityController.Resource)
		observabilityRoutes.GET("/resources/:resourceType/:resourceID/metrics", observabilityController.Metrics)
		observabilityRoutes.GET("/metrics", observabilityController.MetricsSummary)
		observabilityRoutes.GET("/logs", observabilityController.Logs)
		observabilityRoutes.GET("/events", observabilityController.Events)
		observabilityRoutes.GET("/alerts", observabilityController.Alerts)
		observabilityRoutes.GET("/dashboards", observabilityController.Dashboards)
		observabilityRoutes.GET("/integrations", observabilityController.Integrations)
	}

	securityController := controllers.NewSecurityController(database.DB)
	securityRoutes := api.Group("/security")
	securityRoutes.Use(
		middleware.SecurityRequestMetadata(),
		middleware.AuthMiddleware(os.Getenv("JWT_SECRET"), database.DB),
		middleware.SecurityRateLimit(120, time.Minute),
	)
	securityTenants := securityRoutes.Group("/tenants/:tenantID/projects/:projectID")
	{
		securityTenants.GET("/overview", securityController.Overview)
		securityTenants.GET("/policies", securityController.ListPolicies)
		securityTenants.POST("/policies", securityController.CreatePolicy)
		securityTenants.GET("/policies/:policyID", securityController.GetPolicy)
		securityTenants.POST("/policies/:policyID/simulate", securityController.SimulatePolicy)
		securityTenants.GET("/audit", securityController.ListAudit)
	}

	storageController := controllers.NewStorageController(storageService)
	storageRoutes := protected.Group("/storage")
	{
		storageRoutes.GET("/volume-types", storageController.VolumeTypes)
		storageRoutes.GET("/overview", storageController.Overview)
		storageRoutes.GET("/volumes", storageController.ListVolumes)
		storageRoutes.POST("/volumes", storageController.CreateVolume)
		storageRoutes.GET("/volumes/:id", storageController.GetVolume)
		storageRoutes.DELETE("/volumes/:id", storageController.DeleteVolume)
		storageRoutes.GET("/operations", storageController.ListOperations)
		storageRoutes.GET("/operations/:id", storageController.GetOperation)
	}

	cloudInstanceController := controllers.NewCloudInstanceController(cloudService)
	cloudAccountController := controllers.NewCloudAccountController(cloudService)
	internalCloud := r.Group("/internal/cloud")
	internalCloud.Use(
		middleware.SecurityRequestMetadata(),
		middleware.AuthMiddleware(os.Getenv("JWT_SECRET"), database.DB),
		middleware.RequirePlatformSuperAdmin(),
		middleware.IPRateLimit(10, time.Minute),
	)
	internalCloud.PUT("/accounts/:tenantID", cloudAccountController.SetOrganizationAccount)
	cloudRoutes := protected.Group("/cloud")
	cloudRoutes.Use(middleware.SecurityRequestMetadata())
	cloudRoutes.GET("/operations", cloudInstanceController.ListOperations)
	cloudRoutes.GET("/operations/:operationID", cloudInstanceController.GetOperation)
	cloudRoutes.GET("/catalog", cloudInstanceController.Catalog)
	cloudRoutes.GET("/security-groups", cloudInstanceController.ListSecurityGroups)
	cloudRoutes.POST("/security-groups", cloudInstanceController.CreateSecurityGroup)
	cloudInstances := cloudRoutes.Group("/instances")
	{
		cloudInstances.GET("/capabilities", cloudInstanceController.Capabilities)
		cloudInstances.GET("", cloudInstanceController.List)
		cloudInstances.POST("", cloudInstanceController.Create)
		cloudInstances.GET("/:instanceID", cloudInstanceController.Get)
		cloudInstances.POST("/:instanceID/:action", cloudInstanceController.Operate)
		cloudInstances.DELETE("/:instanceID", cloudInstanceController.Delete)
	}

	// -------------------------
	// BLOG ROUTES
	// -------------------------
	blog := protected.Group("/blog")
	{
		blog.POST("/", middleware.RequirePlatformSuperAdmin(), blogController.Create)
		blog.PUT("/", middleware.RequirePlatformSuperAdmin(), blogController.Update)
		blog.DELETE("/:id", middleware.RequirePlatformSuperAdmin(), blogController.Delete)
		blog.GET("/:id", blogController.GetByID)
		blog.GET("/", blogController.GetAll)
	}

	// -------------------------
	// FILE ROUTES
	// -------------------------
	fileController := files.NewController(fileService)

	filesGroup := protected.Group("/files")
	filesGroup.Use(middleware.SecurityRequestMetadata(), middleware.SecurityRateLimit(120, time.Minute))
	{
		filesGroup.POST("/upload", fileController.Upload)
		filesGroup.GET("/objects", fileController.List)
		filesGroup.POST("/direct-uploads", fileController.AuthorizeUpload)
		filesGroup.POST("/direct-uploads/:id/complete", fileController.CompleteUpload)
		filesGroup.GET("/:id/download", fileController.Download)
		filesGroup.GET("/:id/download-url", fileController.Presign)
		filesGroup.DELETE("/:id", fileController.Delete)
	}
	platformAdmin.GET("/health/storage", fileController.Health)

	tenantController := controllers.NewTenantController(database.DB)
	tenants := protected.Group("/tenants")
	{
		tenants.GET("", tenantController.List)
		tenants.POST("", tenantController.Create)
		tenants.GET("/:tenantID/projects", tenantController.ListProjects)
		tenants.POST("/:tenantID/projects", tenantController.CreateProject)
	}

	containerController := controllers.NewContainerDeploymentController(containerService)
	registryController := controllers.NewContainerRegistryController(containerService)
	registries := protected.Group("/container-registries")
	{
		registries.GET("", registryController.List)
		registries.POST("", registryController.Create)
	}
	deployments := protected.Group("/container-deployments")
	{
		deployments.POST("", containerController.Create)
		deployments.GET("", containerController.List)
		deployments.GET("/:deploymentID", containerController.Get)
		deployments.PATCH("/:deploymentID", containerController.Patch)
		deployments.DELETE("/:deploymentID", containerController.Operate)
		deployments.POST("/:deploymentID/:operation", containerController.Operate)
		deployments.GET("/:deploymentID/events", containerController.Events)
		deployments.GET("/:deploymentID/revisions", containerController.Revisions)
		deployments.GET("/:deploymentID/logs", containerController.Logs)
		deployments.GET("/:deploymentID/metrics", containerController.Metrics)
		deployments.GET("/:deploymentID/instances", containerController.Instances)
		deployments.GET("/:deploymentID/configuration", containerController.Configuration)
	}
}
