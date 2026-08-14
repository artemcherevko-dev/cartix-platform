package server

import (
	"context"
	"profile/internal/app/database"
	profilepb "proto/profile"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo *database.Repository
}

func NewService(repo *database.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateProfile(ctx context.Context, userID uuid.UUID, firstName, middleName, lastName string, dateOfBirth time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}

	profile := database.Profile{
		ID:         id,
		UserID:     userID,
		FirstName:  firstName,
		MiddleName: middleName,
		LastName:   lastName,

		DateOfBirth: dateOfBirth,
	}

	if _, err := s.repo.CreateProfile(ctx, profile); err != nil {
		return err
	}
	return nil
}

func (s *Service) GetProfile(ctx context.Context, req *profilepb.GetProfileReq) (*profilepb.ProfileRes, error) {
	profile, err := s.repo.GetProfile(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	return &profilepb.ProfileRes{
		Id:             profile.ID.String(),
		UserId:         profile.UserID.String(),
		FirstName:      profile.FirstName,
		MiddleName:     profile.MiddleName,
		LastName:       profile.LastName,
		DateOfBirthday: profile.DateOfBirth.String(),
	}, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, data database.UpdateDTO) (*profilepb.ProfileRes, error) {
	profile, err := s.repo.UpdateProfile(ctx, uuid.MustParse(userID), data)
	if err != nil {
		return nil, err
	}

	return &profilepb.ProfileRes{
		Id:             profile.ID.String(),
		UserId:         profile.UserID.String(),
		FirstName:      profile.FirstName,
		MiddleName:     profile.MiddleName,
		LastName:       profile.LastName,
		DateOfBirthday: profile.DateOfBirth.String(),
	}, nil
}
