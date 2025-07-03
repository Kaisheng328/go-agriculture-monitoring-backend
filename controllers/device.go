package controllers

import (
	"net/http"
	"time"

	"fyp/config"
	"fyp/utils"

	"github.com/gin-gonic/gin"
)

// GET /device-config/esp32-001
func GetDeviceConfig(c *gin.Context) {
	const duration = 1 * 24 * time.Hour // 14 days
	// const duration = 5 * time.Minute //5minutes
	now := time.Now()
	devMode := false

	if config.DeveloperMode {
		if now.Sub(config.DeveloperModeStart) < duration {
			devMode = true
		} else {
			// Developer mode duration has passed, reset it and re-enable AI
			config.DeveloperMode = false
			config.DeveloperModeStart = time.Time{}
			utils.SetGlobalAIEnabled(true, "Hebe andersonii") // Placeholder plant name
		}
	}

	aiEnabled, _ := utils.IsGlobalAIEnabled()
	c.JSON(http.StatusOK, gin.H{
		"developer_mode":  devMode,
		"start_timestamp": config.DeveloperModeStart.Unix(),
		"ai_enabled":      aiEnabled,
	})
}

// POST /device-config/esp32-001/trigger-dev
func TriggerDeveloperMode(c *gin.Context) {
	config.DeveloperMode = true
	config.DeveloperModeStart = time.Now()
	utils.SetGlobalAIEnabled(false, "") // Disable AI

	c.JSON(http.StatusOK, gin.H{
		"message":         "Developer mode activated for 14 days. AI disabled.",
		"start_timestamp": config.DeveloperModeStart.Unix(),
		"ai_enabled":      false,
	})
}

func StopDeveloperMode(c *gin.Context) {
	config.DeveloperMode = false
	config.DeveloperModeStart = time.Time{}
	utils.SetGlobalAIEnabled(true, "Hebe andersonii") // Re-enable AI with placeholder plant name

	c.JSON(http.StatusOK, gin.H{
		"message":    "✅ Developer mode has been stopped manually. AI enabled.",
		"ai_enabled": true,
	})
}
