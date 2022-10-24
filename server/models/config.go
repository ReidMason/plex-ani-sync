package models

import "gorm.io/gorm"

type Config struct {
	gorm.Model           `gorm:"check:single_record,1 = id"`
	PlexBaseUrl          string `gorm:"not null"`
	PlexToken            string `gorm:"not null"`
	SyncDaysUntilPaused  int    `gorm:"not null"`
	SyncDaysUntilDropped int    `gorm:"not null"`
}
