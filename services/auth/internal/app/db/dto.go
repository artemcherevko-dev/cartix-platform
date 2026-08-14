package db

import "github.com/google/uuid"

type RegisterDTO struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginDTO struct {
	Phone    string `json:"phone,omitempty"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthSessionDTO struct {
	UserID       uuid.UUID `json:"user_id"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"user_agent"`
	RefreshToken string    `json:"refresh_token"`
}
