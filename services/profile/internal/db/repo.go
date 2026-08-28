package db

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateProfile(ctx context.Context, profile Profile) (*Profile, error) {
	if err := r.db.WithContext(ctx).Create(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *Repository) GetProfile(ctx context.Context, id string) (*Profile, error) {
	var profile Profile
	if err := r.db.WithContext(ctx).First(&profile, "user_id = ?", id).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *Repository) UpdateProfile(
	ctx context.Context,
	userID uuid.UUID,
	data UpdateDTO,
) (*Profile, error) {
	updates := make(map[string]any)

	if data.FirstName != nil {
		updates["first_name"] = *data.FirstName
	}

	if data.MiddleName != nil {
		updates["middle_name"] = *data.MiddleName
	}

	if data.LastName != nil {
		updates["last_name"] = *data.LastName
	}

	if data.DateOfBirth != nil {
		updates["date_of_birth"] = *data.DateOfBirth
	}

	var profile Profile

	err := r.db.WithContext(ctx).
		Model(&profile).
		Where("user_id = ?", userID).
		Clauses(clause.Returning{}).
		Updates(updates).Error

	if err != nil {
		return nil, err
	}

	if profile.ID == uuid.Nil {
		return nil, gorm.ErrRecordNotFound
	}

	return &profile, nil
}
