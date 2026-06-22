package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Origin represents registered origins that are allowed to access LumbungFS
type Origin struct {
	ID        string `gorm:"type:varchar(36);primaryKey;" json:"id"`
	Domain    string `gorm:"type:varchar(255);uniqueIndex;not null;" json:"domain"`
	IsBlocked bool   `gorm:"not null;default:false;" json:"is_blocked"`
	ApiKey    string `gorm:"type:varchar(255);" json:"api_key"`
}

// BeforeCreate hook to generate UUIDv7 before inserting
func (o *Origin) BeforeCreate(tx *gorm.DB) (err error) {
	if o.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		o.ID = id.String()
	}
	return nil
}
