package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Connect() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("data/data.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	return db
}
