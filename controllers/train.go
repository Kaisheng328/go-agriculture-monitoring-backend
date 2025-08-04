package controllers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"fyp/config"
	"fyp/models"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func TrainModel(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.TrainModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate plant name
	if req.PlantName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Plant name is required"})
		return
	}

	// Get user info for authorization
	var user models.User
	config.DB.First(&user, userID)

	// Get sensor data for CSV
	query := config.DB.Order("timestamp desc")
	if user.Role != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	var records []models.SensorData
	if err := query.Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sensor data"})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No sensor data available for training"})
		return
	}

	// Create CSV data in memory
	csvData, err := createCSVData(records)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create CSV data"})
		return
	}

	// Send data to Python training service
	response, err := sendTrainingRequest(req.PlantName, csvData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to train model",
			"details": err.Error(),
		})
		return
	}

	// Return the training response
	c.JSON(http.StatusOK, gin.H{
		"message":       "Model training completed",
		"plant_name":    req.PlantName,
		"training_data": response,
	})
}

// createCSVData converts sensor records to CSV format
func createCSVData(records []models.SensorData) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write CSV header
	if err := writer.Write([]string{"timestamp", "temperature", "humidity", "soil_moisture"}); err != nil {
		return nil, err
	}

	// Write data rows
	for _, record := range records {
		row := []string{
			record.Timestamp.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.2f", record.Temperature),
			fmt.Sprintf("%.2f", record.Humidity),
			fmt.Sprintf("%.2f", record.SoilMoisture),
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// sendTrainingRequest sends the plant name and CSV data to Python training service
func sendTrainingRequest(plantName string, csvData []byte) (*models.TrainModelResponse, error) {
	// Python training service URL (adjust as needed)
	pythonServiceURL := os.Getenv("PYTHON_TRAINING_SERVICE_URL")
	if pythonServiceURL == "" {
		pythonServiceURL = "http://localhost:5000" // Default URL
	}
	trainURL := fmt.Sprintf("%s/train", pythonServiceURL)
	// Create multipart form data
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add plant name field
	if err := writer.WriteField("plant_name", plantName); err != nil {
		return nil, fmt.Errorf("failed to write plant_name field: %v", err)
	}

	// Add CSV file
	part, err := writer.CreateFormFile("csv_file", "sensor_data.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}

	if _, err := io.Copy(part, bytes.NewReader(csvData)); err != nil {
		return nil, fmt.Errorf("failed to copy CSV data: %v", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", trainURL, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request with timeout
	client := &http.Client{
		Timeout: 10 * time.Minute, // Model training can take time
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("training service returned error: %s", string(body))
	}

	// Parse response
	var response models.TrainModelResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return &response, nil
}

// GetTrainingStatus checks if a model exists for a specific plant
func GetTrainingStatus(c *gin.Context) {
	plantName := c.Param("plant_name")
	if plantName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Plant name is required"})
		return
	}

	// Check if model exists by calling Python service
	pythonServiceURL := os.Getenv("PYTHON_TRAINING_SERVICE_URL")
	if pythonServiceURL == "" {
		pythonServiceURL = "http://localhost:5000"
	}

	statusURL := fmt.Sprintf("%s/model/status/%s", pythonServiceURL, plantName)

	resp, err := http.Get(statusURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check model status"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read status response"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusNotFound, gin.H{
			"exists":     false,
			"plant_name": plantName,
			"message":    "Model not found",
		})
		return
	}

	var statusResponse map[string]interface{}
	if err := json.Unmarshal(body, &statusResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse status response"})
		return
	}

	c.JSON(http.StatusOK, statusResponse)
}

// ListAvailableModels returns a list of all trained models
func ListAvailableModels(c *gin.Context) {
	pythonServiceURL := os.Getenv("PYTHON_TRAINING_SERVICE_URL")
	if pythonServiceURL == "" {
		pythonServiceURL = "http://localhost:5000"
	}

	modelsURL := fmt.Sprintf("%s/models", pythonServiceURL)

	resp, err := http.Get(modelsURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get models list"})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read models response"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve models"})
		return
	}

	var modelsResponse map[string]interface{}
	if err := json.Unmarshal(body, &modelsResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse models response"})
		return
	}

	c.JSON(http.StatusOK, modelsResponse)
}
