package controllers

import (
	"net/http"
	"time"

	"fyp/config"

	"github.com/gin-gonic/gin"
)

// GET /device-config/esp32-001
func GetDeviceConfig(c *gin.Context) {
	const duration = 14 * 24 * time.Hour // 14 days
	// const duration = 5 * time.Minute //5minutes
	now := time.Now()

	devMode := false
	if config.DeveloperMode && now.Sub(config.DeveloperModeStart) < duration {
		devMode = true
	}

	c.JSON(http.StatusOK, gin.H{
		"developer_mode":  devMode,
		"start_timestamp": config.DeveloperModeStart.Unix(),
	})
}

// POST /device-config/esp32-001/trigger-dev
func TriggerDeveloperMode(c *gin.Context) {
	config.DeveloperMode = true
	config.DeveloperModeStart = time.Now()

	c.JSON(http.StatusOK, gin.H{
		"message":         "Developer mode activated for 14 days",
		"start_timestamp": config.DeveloperModeStart.Unix(),
	})
}
func StopDeveloperMode(c *gin.Context) {
	config.DeveloperMode = false
	config.DeveloperModeStart = time.Time{}

	c.JSON(http.StatusOK, gin.H{
		"message": "✅ Developer mode has been stopped manually",
	})
}
