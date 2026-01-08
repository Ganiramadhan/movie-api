package models

import "time"

type Language struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"uniqueIndex;not null;size:10" json:"code"` // ISO 639-1 code (e.g., 'en', 'id')
	Name      string    `gorm:"not null" json:"name"`                     // Full name (e.g., 'English', 'Indonesian')
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Language) TableName() string {
	return "languages"
}
