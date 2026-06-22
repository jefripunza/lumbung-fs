package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UnknownOrigin logs unregistered domains requesting access
type UnknownOrigin struct {
	ID        string    `gorm:"type:varchar(36);primaryKey;" json:"id"`
	Domain    string    `gorm:"type:varchar(255);not null;" json:"domain"`
	AccessAt  time.Time `gorm:"not null;" json:"access_at"`
	IPAddress string    `gorm:"type:varchar(45);not null;" json:"ip_address"`
}

// BeforeCreate hook to generate UUIDv7 before inserting
func (uo *UnknownOrigin) BeforeCreate(tx *gorm.DB) (err error) {
	if uo.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		uo.ID = id.String()
	}
	if uo.AccessAt.IsZero() {
		uo.AccessAt = time.Now()
	}
	return nil
}
