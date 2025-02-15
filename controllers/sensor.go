package controllers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"fyp/config"
	"fyp/models"
	"fyp/utils"

	"github.com/gin-gonic/gin"
)

// ReceiveData processes incoming sensor data.
func ReceiveData(c *gin.Context) {
	var data models.SensorData
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data"})
		return
	}

	loc, _ := time.LoadLocation("Asia/Kuala_Lumpur")
	data.Timestamp = time.Now().In(loc)

	// Convert userID to uint (handle float64 if needed)
	switch v := userID.(type) {
	case float64:
		data.UserID = uint(v)
	case uint:
		data.UserID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	// Use AI prediction if soil moisture is out of range
	if data.SoilMoisture < 5 || data.SoilMoisture > 95 {
		predicted, err := utils.GetPredictedSoilMoisture(float32(data.Temperature), float32(data.Humidity))
		if err == nil {
			fmt.Println("🔮 Using AI Predicted Soil Moisture:", predicted)
			data.SoilMoisture = float32(predicted)
		} else {
			fmt.Println("❌ AI Prediction failed, keeping original value.")
		}
	}

	data.IsAbnormal = utils.CheckAbnormality(data)
	config.DB.Create(&data)

	// Broadcast data updates
	BroadcastUpdate(data)
	if data.IsAbnormal {
		BroadcastNotification(data)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data received successfully"})
}

// GetHistory returns sensor data history.
func GetHistory(c *gin.Context) {
	var records []models.SensorData
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	config.DB.First(&user, userID)
	if user.Role == "admin" {
		config.DB.Order("timestamp desc").Find(&records)
	} else {
		config.DB.Where("user_id = ?", userID).Order("timestamp desc").Find(&records)
	}
	c.JSON(http.StatusOK, records)
}

// GetAbnormalCount returns the count of abnormal sensor data.
func GetAbnormalCount(c *gin.Context) {
	var count int64
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	config.DB.First(&user, userID)
	if user.Role == "admin" {
		config.DB.Model(&models.SensorData{}).Where("is_abnormal = ?", true).Count(&count)
	} else {
		config.DB.Model(&models.SensorData{}).Where("is_abnormal = ? AND user_id = ?", true, userID).Count(&count)
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetAbnormalHistory returns abnormal sensor data records.
func GetAbnormalHistory(c *gin.Context) {
	var records []models.SensorData
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	config.DB.First(&user, userID)
	query := config.DB.Where("is_abnormal = ?", true)
	if user.Role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Order("timestamp desc").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var response []map[string]interface{}
	for _, record := range records {
		response = append(response, map[string]interface{}{
			"timestamp": record.Timestamp.Format("2006-01-02 15:04:05"),
			"type":      utils.GetAbnormalType(record),
		})
	}

	c.JSON(http.StatusOK, response)
}

// DownloadCSV sends sensor data as a CSV file.
func DownloadCSV(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	config.DB.First(&user, userID)
	query := config.DB.Order("timestamp desc")
	if user.Role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	var records []models.SensorData
	query.Find(&records)

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=sensor_data.csv")
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	writer.Write([]string{"timestamp", "temperature", "humidity", "soil_moisture"})
	for _, record := range records {
		writer.Write([]string{
			record.Timestamp.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.2f", record.Temperature),
			fmt.Sprintf("%.2f", record.Humidity),
			fmt.Sprintf("%.2f", record.SoilMoisture),
		})
	}
}

// DeleteRecord deletes a single sensor data record.
func DeleteRecord(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	var record models.SensorData
	if err := config.DB.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	var user models.User
	config.DB.First(&user, userID)
	if err := config.DB.Delete(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record deleted successfully"})
}

// DeleteAllRecords deletes all sensor data records (admin only).
func DeleteAllRecords(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user models.User
	config.DB.First(&user, userID)
	if user.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to delete all records"})
		return
	}

	if err := config.DB.Exec("DELETE FROM sensor_data").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete records"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All records deleted successfully"})
}
