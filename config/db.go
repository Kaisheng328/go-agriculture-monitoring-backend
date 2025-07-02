package config

import (
	"time"

	"gorm.io/gorm"
)

// DB is a global variable to hold the database connection
var DB *gorm.DB

var DeveloperMode bool = false
var DeveloperModeStart time.Time
