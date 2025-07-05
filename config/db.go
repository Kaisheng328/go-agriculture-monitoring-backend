package config

import (
	"fyp/models"
	"sync"
	"time"

	"gorm.io/gorm"
)

// DB is a global variable to hold the database connection
var DB *gorm.DB

// developerModeStateCache holds the current developer mode state in memory
// and is synchronized with the database.
type developerModeStateCache struct {
	IsEnabled bool
	StartTime time.Time
}

var (
	currentDevModeState developerModeStateCache
	devModeMutex        sync.Mutex
)

const developerModeSettingID = 1 // Assuming a single global setting

// InitDeveloperModeState loads the developer mode state from the database
// or creates a default entry if one doesn't exist.
// This should be called on application startup.
func InitDeveloperModeState(db *gorm.DB) error {
	devModeMutex.Lock()
	defer devModeMutex.Unlock()

	var setting models.DeveloperModeSetting
	result := db.First(&setting, developerModeSettingID)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			// Create a default setting
			setting = models.DeveloperModeSetting{
				ID:        developerModeSettingID,
				IsEnabled: false,
				StartTime: time.Time{}, // Zero time
			}
			if err := db.Create(&setting).Error; err != nil {
				return err // Failed to create default setting
			}
		} else {
			return result.Error // Other database error
		}
	}

	currentDevModeState.IsEnabled = setting.IsEnabled
	currentDevModeState.StartTime = setting.StartTime
	return nil
}

// GetDeveloperModeState returns the current cached developer mode state.
func GetDeveloperModeState() (isEnabled bool, startTime time.Time) {
	devModeMutex.Lock()
	defer devModeMutex.Unlock()
	return currentDevModeState.IsEnabled, currentDevModeState.StartTime
}

// SetDeveloperModeState updates the developer mode state in both the database and the cache.
func SetDeveloperModeState(db *gorm.DB, isEnabled bool, startTime time.Time) error {
	devModeMutex.Lock()
	defer devModeMutex.Unlock()

	setting := models.DeveloperModeSetting{
		ID:        developerModeSettingID,
		IsEnabled: isEnabled,
		StartTime: startTime,
	}

	// Use Save to update or create if somehow missing (though Init should prevent this)
	if err := db.Save(&setting).Error; err != nil {
		return err
	}

	currentDevModeState.IsEnabled = isEnabled
	currentDevModeState.StartTime = startTime
	return nil
}
