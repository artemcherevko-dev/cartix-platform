package db

import (
	"time"

	"github.com/google/uuid"
)

type UserStatus string
type UserRole string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusPending  UserStatus = "pending"
	UserStatusInactive UserStatus = "inactive"
)

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

type User struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	Email         string     `json:"email" gorm:"uniqueIndex;size:255;not null"`
	Phone         string     `json:"phone" gorm:"uniqueIndex;size:25;not null"`
	PasswordHash  string     `json:"-" gorm:"not null"`
	EmailVerified bool       `json:"email_verified" gorm:"not null;default:false"`
	Status        UserStatus `json:"status" gorm:"type:varchar(20);not null;default:'pending'"`

	Sessions []AuthSession `json:"sessions,omitempty" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	Role     UserRole      `json:"role" gorm:"type:varchar(20);default:'user';not null"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuthSession struct {
	ID     uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `json:"user_id" gorm:"type:uuid;index;not null"`

	RefreshTokenHash string `json:"-" gorm:"size:64;uniqueIndex;not null"`
	UserAgent        string `json:"user_agent" gorm:"type:text"`
	IP               string `json:"ip" gorm:"size:45"`

	ExpiresAt time.Time `json:"expires_at" gorm:"index"`
	CreatedAt time.Time `json:"created_at"`
}
