package controllers

import (
	"fyp/config"
	"fyp/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const developerModeDuration = 14 * 24 * time.Hour // 14 days
// const developerModeDuration = 5 * time.Minute // 5 minutes for testing

// GET /device-config/esp32-001
func GetDeviceConfig(c *gin.Context) {
	currentDevModeEnabled, currentDevModeStartTime := config.GetDeveloperModeState()
	now := time.Now()
	devModeActive := false
	var responseStartTime time.Time = currentDevModeStartTime // Use this for the JSON response

	if currentDevModeEnabled {
		if now.Sub(currentDevModeStartTime) < developerModeDuration {
			devModeActive = true
		} else {
			// Developer mode duration has passed, reset it and re-enable AI
			err := config.SetDeveloperModeState(config.DB, false, time.Time{})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update developer mode state"})
				return
			}
			utils.SetGlobalAIEnabled(true, "Hebe andersonii") // Placeholder plant name
			responseStartTime = time.Time{}                   // Reflect that it's now off
		}
	}

	aiEnabled, _ := utils.IsGlobalAIEnabled()
	c.JSON(http.StatusOK, gin.H{
		"developer_mode":  devModeActive,
		"start_timestamp": responseStartTime.Unix(), // Use Unix timestamp of responseStartTime
		"ai_enabled":      aiEnabled,
	})
}

// POST /device-config/esp32-001/trigger-dev
func TriggerDeveloperMode(c *gin.Context) {
	startTime := time.Now()
	err := config.SetDeveloperModeState(config.DB, true, startTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate developer mode"})
		return
	}
	utils.SetGlobalAIEnabled(false, "") // Disable AI

	c.JSON(http.StatusOK, gin.H{
		"message":         "Developer mode activated for 14 days. AI disabled.",
		"start_timestamp": startTime.Unix(),
		"ai_enabled":      false,
	})
}

// POST /device-config/esp32-001/stop-dev (Note: Changed from GET in original plan for consistency with Trigger)
func StopDeveloperMode(c *gin.Context) {
	err := config.SetDeveloperModeState(config.DB, false, time.Time{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop developer mode"})
		return
	}
	utils.SetGlobalAIEnabled(true, "Hebe andersonii") // Re-enable AI with placeholder plant name

	c.JSON(http.StatusOK, gin.H{
		"message":    "✅ Developer mode has been stopped manually. AI enabled.",
		"ai_enabled": true,
	})
}
