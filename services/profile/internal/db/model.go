package db

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID     uuid.UUID `gorm:"primary_key"`
	UserID uuid.UUID `gorm:"not null;uniqueIndex"`

	FirstName   string    `gorm:"not null"`
	MiddleName  string    `gorm:"not null"`
	LastName    string    `gorm:"not null"`
	DateOfBirth time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`

	CreatedAt time.Time `gorm:"not null"`
}
