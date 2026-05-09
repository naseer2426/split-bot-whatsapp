package db

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init establishes a GORM connection to Postgres using the given DSN.
func Init(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("database DSN is empty")
	}
	if DB != nil {
		return DB, nil
	}

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	DB = database
	return DB, nil
}

// GetDB returns the global DB handle. Panics if [Init] was not called successfully first.
func GetDB() *gorm.DB {
	if DB == nil {
		panic("db not initialized; call Init with a non-empty DSN first")
	}
	return DB
}
