package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PresignedURL struct {
	ID        string    `gorm:"type:varchar(36);primaryKey;" json:"id"`
	OriginID  string    `gorm:"type:varchar(36);not null;index;" json:"origin_id"`
	Path      string    `gorm:"type:varchar(255);not null;" json:"path"`
	Token     string    `gorm:"type:varchar(255);uniqueIndex;not null;" json:"token"`
	CreatedAt time.Time `gorm:"not null;" json:"created_at"`
}

func (p *PresignedURL) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		p.ID = id.String()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	return nil
}
