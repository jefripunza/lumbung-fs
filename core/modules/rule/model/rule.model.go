package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Rule defines routing validation and restriction rules for files under specific paths
type Rule struct {
	ID                  string `gorm:"type:varchar(36);primaryKey;" json:"id"`
	OriginID            string `gorm:"type:varchar(36);not null;index;" json:"origin_id"`
	Path                string `gorm:"type:varchar(255);not null;" json:"path"` // e.g. "ktp" for "/file/.../ktp"
	ValidateMethod      string `gorm:"type:varchar(255);" json:"validate_method"` // e.g. "jwt", "headers", "cookies"
	ValidateHeaders     string `gorm:"type:varchar(512);" json:"validate_headers"` // comma-separated required headers
	ValidateURL         string `gorm:"type:varchar(1024);" json:"validate_url"` // external verification endpoint
	ValidateFallbackURL string `gorm:"type:varchar(1024);" json:"validate_fallback_url"` // fallback redirect URL
	IsMaxSize           bool   `gorm:"not null;default:false;" json:"is_max_size"`
	ValueMaxSize        int    `gorm:"not null;default:0;" json:"value_max_size"`
	ValueUnitSize       string `gorm:"type:varchar(10);not null;default:'MB';" json:"value_unit_size"` // KB, MB, GB
	IsExtensions        bool   `gorm:"not null;default:false;" json:"is_extensions"`
	ValueExtensions     string `gorm:"type:varchar(512);" json:"value_extensions"` // comma-separated, e.g. "png,jpg,jpeg"
	IsCompress          bool   `gorm:"not null;default:false;" json:"is_compress"`
	CompressLevel       int    `gorm:"not null;default:3;" json:"compress_level"`
	IsEncrypt           bool   `gorm:"not null;default:false;" json:"is_encrypt"`
	EncryptionKey       string `gorm:"type:varchar(255);" json:"encryption_key"`
}

// BeforeCreate hook to generate UUIDv7 before inserting
func (r *Rule) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		r.ID = id.String()
	}
	return nil
}
