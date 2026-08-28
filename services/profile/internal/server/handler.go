package server

import (
	"context"
	"profile/internal/db"
	profilepb "proto/profile"
)

type Handler struct {
	profilepb.UnimplementedProfileServer
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetProfile(ctx context.Context, req *profilepb.GetProfileReq) (*profilepb.ProfileRes, error) {
	return h.service.GetProfile(ctx, req)
}

func (h *Handler) UpdateProfile(ctx context.Context, req *profilepb.UpdateProfileReq) (*profilepb.ProfileRes, error) {
	data := db.UpdateDTO{
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		MiddleName:  req.MiddleName,
		DateOfBirth: req.DateOfBirth,
	}

	profile, err := h.service.UpdateProfile(ctx, req.UserId, data)
	if err != nil {
		return nil, err
	}

	return profile, nil
}
