package controllers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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
	if err := config.DB.Exec("ALTER SEQUENCE sensor_data_id_seq RESTART WITH 1").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset primary key sequence"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "All records deleted successfully"})
}

// Update edit a  sensor data record.
func UpdateRecord(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	var record models.SensorData

	// Check if the record exists
	if err := config.DB.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	// Bind the JSON input to a temporary struct
	var input models.SensorData
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Update record fields
	record.Temperature = input.Temperature
	record.Humidity = input.Humidity
	record.SoilMoisture = input.SoilMoisture

	// Save changes to the database
	var user models.User
	config.DB.First(&user, userID)
	if err := config.DB.Save(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Record updated successfully", "updated_record": record})
}

func HandleDeviceLocation(c *gin.Context) {
	var payload models.GeolocationRequest

	// Validate incoming JSON payload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
		return
	}

	// Extract userID from JWT (handle both float64 and uint types)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Convert userID to uint safely
	var userIDUint uint
	switch v := userID.(type) {
	case float64:
		userIDUint = uint(v)
	case uint:
		userIDUint = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	// Send Wi-Fi data to Google API
	location, err := GetLocationFromGoogle(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get location"})
		return
	}

	// Store location in the database
	deviceLocation := models.DeviceLocation{
		DeviceID:  userIDUint, // Use converted user ID
		Latitude:  location.Location.Lat,
		Longitude: location.Location.Lng,
		Accuracy:  location.Accuracy,
	}

	if err := config.DB.Create(&deviceLocation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store location"})
		return
	}

	// Return the location to the client
	c.JSON(http.StatusOK, gin.H{
		"message":   "Location stored successfully",
		"latitude":  location.Location.Lat,
		"longitude": location.Location.Lng,
		"accuracy":  location.Accuracy,
	})
}

// GetDeviceLocation: Retrieves the latest location by user_id
func GetDeviceLocation(c *gin.Context) {
	deviceID := c.Param("device_id")

	var location models.DeviceLocation

	// Find the latest location by device_id
	if err := config.DB.Where("device_id = ?", deviceID).Order("timestamp DESC").First(&location).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Location not found"})
		return
	}

	// Return the location details
	c.JSON(http.StatusOK, gin.H{
		"device_id": location.DeviceID,
		"latitude":  location.Latitude,
		"longitude": location.Longitude,
		"accuracy":  location.Accuracy,
	})
}

// GetLocationFromGoogle: Calls Google Geolocation API
func GetLocationFromGoogle(data models.GeolocationRequest) (models.GeolocationResponse, error) {
	var geoResp models.GeolocationResponse
	apiKey := os.Getenv("GOOGLE_API_KEY")
	url := fmt.Sprintf("https://www.googleapis.com/geolocation/v1/geolocate?key=%s", apiKey)

	jsonData, _ := json.Marshal(data)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return geoResp, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &geoResp)

	return geoResp, nil
}
