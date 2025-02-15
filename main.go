package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"fyp/config"
	"fyp/controllers"
	"fyp/middlewares"
)

func main() {
	// Load environment variables
	godotenv.Load()

	// Connect to PostgreSQL database
	dsn := os.Getenv("DATABASE_URL")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	// Set the global DB in the config package and migrate models
	config.DB = db
	controllers.MigrateModels(db)

	// Set up Gin router with CORS configuration
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://fyp-backend-bd5cc.web.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	// Public routes
	r.POST("/signup", controllers.Signup)
	r.POST("/login", controllers.Login)

	// Protected routes using auth middleware
	auth := r.Group("/")
	auth.Use(middlewares.AuthMiddleware())
	auth.POST("/promote-admin", controllers.PromoteToAdmin)
	auth.POST("/sensor-data", controllers.ReceiveData)
	auth.GET("/history", controllers.GetHistory)
	auth.GET("/abnormal-count", controllers.GetAbnormalCount)
	auth.GET("/abnormal-history", controllers.GetAbnormalHistory)
	auth.GET("/download-csv", controllers.DownloadCSV)
	auth.GET("/ws", controllers.HandleWebSocket)
	auth.DELETE("/delete/:id", controllers.DeleteRecord)
	auth.DELETE("/delete/all", controllers.DeleteAllRecords)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
