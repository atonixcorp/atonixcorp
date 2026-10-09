package main

import (
	"atonixcorp/api/internal/config"
	"atonixcorp/api/internal/controllers"
	"atonixcorp/api/internal/database"
	"atonixcorp/api/internal/repositories"
	"atonixcorp/api/internal/routes"
	"atonixcorp/api/internal/services"

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
	routes.SetupRoutes(r, authController, contactController, blogController)

	r.Run(":8080")
}
