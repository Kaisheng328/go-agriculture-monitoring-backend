package main

import (
	"log"
	"os"

	"fyp/config"
	"fyp/controllers"
	"fyp/middlewares"
	"fyp/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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
	controllers.MigrateModels(db) // This will migrate User, SensorData, and DeveloperModeSetting
	config.DB.AutoMigrate(&models.DeviceLocation{})

	// Initialize developer mode state from DB
	if err := config.InitDeveloperModeState(config.DB); err != nil {
		log.Fatalf("Failed to initialize developer mode state: %v", err)
	}

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
	r.POST("/toggle-ai", controllers.ToggleAI)

	// Protected routes using auth middleware
	auth := r.Group("/")
	auth.Use(middlewares.AuthMiddleware())
	auth.GET("/ws", controllers.HandleWebSocket)
	auth.POST("/promote-admin", controllers.PromoteToAdmin)
	auth.POST("/promote-user", controllers.PromoteToUser)
	auth.POST("/sensor-data", controllers.ReceiveData)
	auth.POST("/device-config/esp32-001/stop-dev", controllers.StopDeveloperMode)
	auth.POST("/device-config/esp32-001/trigger-dev", controllers.TriggerDeveloperMode)
	auth.GET("/history", controllers.GetHistory)
	auth.GET("/users", controllers.GetUsers)
	auth.GET("/profile", controllers.GetProfile)
	auth.GET("/abnormal-count", controllers.GetAbnormalCount)
	auth.GET("/abnormal-history", controllers.GetAbnormalHistory)
	auth.GET("/download-csv", controllers.DownloadCSV)
	auth.GET("/device-config/esp32-001", controllers.GetDeviceConfig)
	auth.PUT("/update/:id", controllers.UpdateRecord)
	auth.DELETE("/delete/:id", controllers.DeleteRecord)
	auth.DELETE("/delete/all", controllers.DeleteAllRecords)
	auth.DELETE("/delete/my-records", controllers.DeleteMyRecords)
	auth.DELETE("/delete/user/:user_id", controllers.DeleteUserRecords)
	auth.DELETE("/admin/delete-user/:user_id", controllers.DeleteUserAccount)
	auth.POST("/location", controllers.HandleDeviceLocation)            // POST location from ESP32
	auth.GET("/get-location/:device_id", controllers.GetDeviceLocation) // GET location for frontend
	auth.POST("/train-model", controllers.TrainModel)
	auth.GET("/model/status/:plant_name", controllers.GetTrainingStatus)
	auth.GET("/models", controllers.ListAvailableModels)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
