package app

import (
	"auth/cmd/internal/app/jwt"
	"auth/cmd/internal/app/nats"
	"auth/cmd/internal/db"
	"auth/cmd/internal/lib"
	"context"
	"errors"
	"log"
	nats2 "nats"
	"regexp"
	"strings"

	authpb "proto/auth"

	"google.golang.org/grpc/metadata"
)

type Handler struct {
	authpb.UnimplementedAuthServer
	repo *db.Repo
	nats *nats2.Client
}

func NewHandler(
	repo *db.Repo,
	natsClient *nats2.Client,
) *Handler {
	return &Handler{
		repo: repo,
		nats: natsClient,
	}
}

func (h *Handler) Register(ctx context.Context, req *authpb.RegisterReq) (*authpb.AuthRes, error) {
	user, err := h.repo.CreateUser(db.RegisterDTO{
		Email:     req.Email,
		Password:  req.Password,
		Phone:     req.Phone,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		return nil, err
	}

	err = nats.PublishUserRegistered(ctx, h.nats, nats.UserRegistered{
		UserID:    user.ID.String(),
		Email:     req.Email,
		Phone:     req.Phone,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		return nil, err
	}

	access, refresh, err := jwt.GenerateTokenPair(user.ID.String(), user.Role)
	if err != nil {
		return nil, err
	}

	ip, userAgent := getSessionMetadata(ctx)
	if ip == "" || userAgent == "" {
		log.Println("[AUTH] Handle Register: failed to get session metadata")
	}

	session := db.AuthSessionDTO{
		UserID:       user.ID,
		IP:           ip,
		UserAgent:    userAgent,
		RefreshToken: refresh,
	}

	err = h.repo.CreateAuthSession(ctx, session)
	if err != nil {
		return nil, err
	}

	return &authpb.AuthRes{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (h *Handler) Login(ctx context.Context, req *authpb.LoginReq) (*authpb.AuthRes, error) {
	phoneRegex := regexp.MustCompile(`^\+?[0-9]{8,15}$`)

	switch {
	case strings.Contains(req.Login, "@"):
		user, err := h.repo.GetUserByEmail(ctx, req.Login)
		if err != nil {
			return nil, err
		}

		ok, err := lib.VerifyPassword([]byte(req.Password), []byte(user.PasswordHash))
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New("invalid password")
		}

		access, refresh, session, err := newAuthSession(ctx, user)
		err = h.repo.CreateAuthSession(ctx, session)
		if err != nil {
			return nil, err
		}

		return &authpb.AuthRes{
			AccessToken:  access,
			RefreshToken: refresh,
		}, nil
	case phoneRegex.MatchString(req.Login):
		user, err := h.repo.GetUserByPhone(ctx, req.Login)
		if err != nil {
			return nil, err
		}

		ok, err := lib.VerifyPassword([]byte(req.Password), []byte(user.PasswordHash))
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errors.New("invalid password")
		}

		access, refresh, session, err := newAuthSession(ctx, user)
		err = h.repo.CreateAuthSession(ctx, session)
		if err != nil {
			return nil, err
		}

		return &authpb.AuthRes{
			AccessToken:  access,
			RefreshToken: refresh,
		}, nil
	}
	return nil, errors.New("invalid login or password")
}

func (h *Handler) Refresh(ctx context.Context, req *authpb.RefreshReq) (*authpb.AuthRes, error) {
	log.Println("[AUTH] Handle Refresh")
	user, session, err := h.repo.GetUserAndSessionByRefreshToken(ctx, lib.Hash(req.RefreshToken))
	if err != nil {
		return nil, err
	}

	access, refresh, err := jwt.GenerateTokenPair(user.ID.String(), user.Role)
	if err != nil {
		return nil, err
	}
	refreshTokenHash := lib.Hash(refresh)

	err = h.repo.UpdateAuthSession(ctx, session.ID, refreshTokenHash)
	if err != nil {
		return nil, err
	}

	return &authpb.AuthRes{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (h *Handler) Validate(ctx context.Context, req *authpb.ValidateTokenReq) (*authpb.ValidateTokenRes, error) {
	claims, err := jwt.ParseAccessToken(req.AccessToken)
	if err != nil {
		return nil, err
	}

	user, err := h.repo.GetUserByID(ctx, claims.Subject)
	if err != nil {
		return nil, err
	}

	active := user.Status == "active"
	return &authpb.ValidateTokenRes{User: &authpb.UserRes{
		Id:    user.ID.String(),
		Email: user.Email,
		Phone: user.Phone,

		EmailVerified: user.EmailVerified,
		PhoneVerified: user.PhoneVerified,
		Role:          string(user.Role),
		Active:        active,
	}}, nil
}

func getSessionMetadata(ctx context.Context) (ip, userAgent string) {
	md, _ := metadata.FromIncomingContext(ctx)

	if values := md.Get("x-client-ip"); len(values) > 0 {
		ip = values[0]
	}

	if values := md.Get("x-user-agent"); len(values) > 0 {
		userAgent = values[0]
	}

	return ip, userAgent
}

func newAuthSession(ctx context.Context, user *db.User) (string, string, db.AuthSessionDTO, error) {

	access, refresh, err := jwt.GenerateTokenPair(user.ID.String(), user.Role)
	if err != nil {
		return "", "", db.AuthSessionDTO{}, err
	}

	ip, userAgent := getSessionMetadata(ctx)
	if ip == "" || userAgent == "" {
		log.Println("[AUTH] Handle Register: failed to get session metadata")
	}

	session := db.AuthSessionDTO{
		UserID:       user.ID,
		IP:           ip,
		UserAgent:    userAgent,
		RefreshToken: refresh,
	}
	return access, refresh, session, nil
}
