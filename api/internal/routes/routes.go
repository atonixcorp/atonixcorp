package routes

import (
	"atonixcorp/api/internal/controllers"
	"atonixcorp/api/internal/database"
	"atonixcorp/api/internal/files"
	"atonixcorp/api/internal/security/middleware"

	"os"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	r *gin.Engine,
	authController *controllers.AuthController,
	contactController *controllers.ContactController,
	blogController *controllers.BlogController,
) {

	// -------------------------
	// GLOBAL MIDDLEWARE
	// -------------------------
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.AuthMiddleware(os.Getenv("JWT_SECRET")))

	// -------------------------
	// API VERSIONING
	// -------------------------
	api := r.Group("/api/v1")

	// -------------------------
	// AUTH ROUTES
	// -------------------------
	auth := api.Group("/auth")
	{
		auth.POST("/signup", authController.Signup)
		auth.POST("/login", authController.Login)
	}

	// -------------------------
	// CONTACT ROUTES
	// -------------------------
	contact := api.Group("/contact")
	{
		contact.POST("/submit", contactController.Submit)
		contact.GET("/messages", contactController.GetAll)
	}

	// -------------------------
	// BLOG ROUTES
	// -------------------------
	blog := api.Group("/blog")
	{
		blog.POST("/", blogController.Create)
		blog.PUT("/", blogController.Update)
		blog.DELETE("/:id", blogController.Delete)
		blog.GET("/:id", blogController.GetByID)
		blog.GET("/", blogController.GetAll)
	}

	// -------------------------
	// FILE ROUTES
	// -------------------------
	fileRepo := files.NewFileRepository(database.DB)
	fileService := files.NewFileService(fileRepo)
	fileController := files.NewFileController(fileService)

	filesGroup := api.Group("/files")
	{
		filesGroup.POST("/upload", fileController.Upload)
		filesGroup.GET("/:id/download", fileController.Download)
	}
}
