package db

import (
	"auth/internal/app/lib"
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repo struct {
	db *gorm.DB
}

func NewRepo(db *gorm.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CreateUser(data RegisterDTO) (*User, error) {
	passwordHash, err := lib.HashPassword(data.Password)

	userID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	user := User{
		ID:           userID,
		Email:        data.Email,
		Phone:        data.Phone,
		PasswordHash: passwordHash,
	}

	if err := r.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repo) GetUserByID(ctx context.Context, id string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *Repo) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *Repo) GetUserByPhone(ctx context.Context, phone string) (*User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repo) CreateAuthSession(ctx context.Context, data AuthSessionDTO) error {
	sessionID, err := uuid.NewV7()
	if err != nil {
		return err
	}
	session := AuthSession{
		ID:               sessionID,
		UserID:           data.UserID,
		RefreshTokenHash: lib.Hash(data.RefreshToken),
		UserAgent:        data.UserAgent,
		IP:               data.IP,
		ExpiresAt:        time.Now().Add(time.Hour * 24),
	}
	if err := r.db.WithContext(ctx).Create(&session).Error; err != nil {
		return err
	}

	return nil
}

func (r *Repo) UpdateAuthSession(ctx context.Context, id uuid.UUID, refreshToken string) error {
	if err := r.db.WithContext(ctx).Model(&AuthSession{}).Where("id = ?", id).Updates(AuthSession{
		RefreshTokenHash: refreshToken,
		ExpiresAt:        time.Now().Add(time.Hour * 24),
		CreatedAt:        time.Now(),
	}).Error; err != nil {
		return err
	}
	return nil
}

func (r *Repo) GetUserAndSessionByRefreshToken(ctx context.Context, refreshToken string) (*User, *AuthSession, error) {
	var user User
	var session AuthSession
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("refresh_token_hash = ?", refreshToken).First(&session).Error; err != nil {
			log.Println(refreshToken)
			return err
		}

		if err := tx.Where("id = ?", session.UserID).First(&user).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, nil, err
	}
	return &user, &session, nil
}
