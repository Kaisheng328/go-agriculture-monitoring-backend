package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db         *gorm.DB
	wsUpgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	clients = make(map[*websocket.Conn]bool)
)

type SensorData struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Timestamp    time.Time `json:"timestamp"`
	Temperature  float64   `json:"temperature"`
	Humidity     float64   `json:"humidity"`
	SoilMoisture float64   `json:"soil_moisture"`
	IsAbnormal   bool      `json:"is_abnormal"`
}

func main() {
	// Load environment variables
	godotenv.Load()

	// Connect to PostgreSQL database
	dsn := os.Getenv("DATABASE_URL")
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	db.AutoMigrate(&SensorData{})

	// Set up Gin router
	r := gin.Default()
	r.Use(cors.Default())
	// Routes
	r.POST("/sensor-data", receiveData)
	r.GET("/history", getHistory)
	r.GET("/abnormal-count", getAbnormalCount)
	r.GET("/abnormal-history", getAbnormalHistory)
	r.GET("/download-csv", downloadCSV)
	r.GET("/ws", handleWebSocket)
	r.DELETE("/delete/:id", deleteRecord)
	r.DELETE("/delete/all", deleteAllRecords)
	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(r.Run(":" + port))
}

func receiveData(c *gin.Context) {
	var data SensorData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
		return
	}
	loc, _ := time.LoadLocation("Asia/Kuala_Lumpur")

	// Assign timestamp with Malaysia timezone
	data.Timestamp = time.Now().In(loc)
	// Just pass the data without changing the timestamp, as PostgreSQL is already handling it
	data.IsAbnormal = checkAbnormality(data)
	db.Create(&data)

	// Broadcast the data update
	broadcastUpdate(data)

	// If the data is abnormal, broadcast a notification
	if data.IsAbnormal {
		broadcastNotification(data)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data received successfully"})
}

func getHistory(c *gin.Context) {
	var records []SensorData
	db.Order("timestamp desc").Find(&records)
	c.JSON(http.StatusOK, records)
}

func getAbnormalCount(c *gin.Context) {
	var count int64
	db.Model(&SensorData{}).Where("is_abnormal = ?", true).Count(&count)
	c.JSON(http.StatusOK, gin.H{"count": count})
}

func getAbnormalHistory(c *gin.Context) {
	var records []SensorData
	// Fetch abnormal records ordered by timestamp in descending order
	if err := db.Where("is_abnormal = ?", true).Order("timestamp desc").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Prepare the response
	var response []map[string]interface{}
	for _, record := range records {
		// Determine the abnormal type
		abnormalType := getAbnormalType(record)

		// Format timestamp to the desired format
		formattedTimestamp := record.Timestamp.Format("2006-01-02 15:04:05")

		// Add the formatted data to the response
		response = append(response, map[string]interface{}{
			"timestamp": formattedTimestamp,
			"type":      abnormalType,
		})
	}

	// Return the response as JSON
	c.JSON(http.StatusOK, response)
}

func getAbnormalType(record SensorData) string {
	if record.Temperature < 20 || record.Temperature > 50 {
		return "Temperature"
	}
	if record.Humidity < 30 || record.Humidity > 90 {
		return "Humidity"
	}
	if record.SoilMoisture < 5 || record.SoilMoisture > 95 {
		return "Soil Moisture"
	}
	return "Unknown"
}

func downloadCSV(c *gin.Context) {
	records := []SensorData{}
	db.Order("timestamp desc").Find(&records)

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=sensor_data.csv")
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	writer.Write([]string{"timestamp", "temperature", "humidity", "soil_moisture"})
	for _, record := range records {
		writer.Write([]string{
			record.Timestamp.Format("2006-01-02 15:04:05"), // Format the timestamp
			fmt.Sprintf("%.2f", record.Temperature),
			fmt.Sprintf("%.2f", record.Humidity),
			fmt.Sprintf("%.2f", record.SoilMoisture),
		})
	}
}
func deleteRecord(c *gin.Context) {
	id := c.Param("id")
	var record SensorData

	// Check if the record exists
	if err := db.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// Delete the record
	if err := db.Delete(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

func deleteAllRecords(c *gin.Context) {
	// Delete all records from the SensorData table
	if err := db.Exec("DELETE FROM sensor_data").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete records"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All records deleted successfully"})
}
func handleWebSocket(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	clients[conn] = true
}

func broadcastUpdate(data SensorData) {
	msg, _ := json.Marshal(data)
	for client := range clients {
		client.WriteMessage(websocket.TextMessage, msg)
	}
}

func broadcastNotification(data SensorData) {
	notification := map[string]interface{}{
		"message": "Abnormal data detected!",
		"data":    data,
	}
	msg, _ := json.Marshal(notification)
	for client := range clients {
		client.WriteMessage(websocket.TextMessage, msg)
	}
}

func checkAbnormality(data SensorData) bool {
	return data.Temperature < 20 || data.Temperature > 50 ||
		data.Humidity < 30 || data.Humidity > 90 ||
		data.SoilMoisture < 5 || data.SoilMoisture > 95
}
