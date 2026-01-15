package db

import "log"

// AutoMigrate runs GORM auto-migrations for all models.
func AutoMigrate() {
	database := GetDB()
	if err := database.AutoMigrate(&SplitBotUser{}); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
}
