package server

import (
	"auth/internal/db"
	"auth/internal/hashing"
	"auth/internal/jwt"
	"auth/internal/nats"
	"auth/internal/verify"
	"context"
	"errors"
	"log"
	natsclient "nats"
	"regexp"
	"strings"
	"time"

	authpb "proto/auth"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type Handler struct {
	authpb.UnimplementedAuthServer
	repo   *db.Repo
	nats   *natsclient.Client
	verify *verify.Store
}

func NewHandler(
	repo *db.Repo,
	natsClient *natsclient.Client,
	verifyStore *verify.Store,
) *Handler {
	return &Handler{
		repo:   repo,
		nats:   natsClient,
		verify: verifyStore,
	}
}

func (h *Handler) Register(ctx context.Context, req *authpb.RegisterReq) (*authpb.AuthRes, error) {
	user, err := h.repo.CreateUser(db.RegisterDTO{
		Email:    req.Email,
		Password: req.Password,
		Phone:    req.Phone,
	})
	if err != nil {
		return nil, err
	}

	dateOfBirth, err := time.Parse("02-01-2006", req.DateOfBirth)
	if err != nil {
		return nil, err
	}

	verifyToken, err := verify.NewVerifyToken()
	if err != nil {
		return nil, err
	}

	if err := h.verify.SaveEmailToken(ctx, verifyToken, user.ID.String()); err != nil {
		return nil, err
	}

	err = nats.PublishUserRegistered(ctx, h.nats, nats.UserRegistered{
		UserID:      user.ID.String(),
		FirstName:   req.FirstName,
		MiddleName:  req.MiddleName,
		LastName:    req.LastName,
		Email:       req.Email,
		VerifyToken: verifyToken,
		DateOfBirth: dateOfBirth,
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

		ok, err := hashing.VerifyPassword([]byte(req.Password), []byte(user.PasswordHash))
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

		ok, err := hashing.VerifyPassword([]byte(req.Password), []byte(user.PasswordHash))
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
	user, session, err := h.repo.GetUserAndSessionByRefreshToken(ctx, hashing.Hash(req.RefreshToken))
	if err != nil {
		return nil, err
	}

	access, refresh, err := jwt.GenerateTokenPair(user.ID.String(), user.Role)
	if err != nil {
		return nil, err
	}
	refreshTokenHash := hashing.Hash(refresh)

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
		Role:          string(user.Role),
		Active:        active,
	}}, nil
}

func (h *Handler) VerifyEmail(ctx context.Context, req *authpb.VerifyEmailReq) (*authpb.VerifyEmailRes, error) {
	if req.Token == "" {
		return nil, verify.ErrInvalidToken
	}

	userID, err := h.verify.ConsumeEmailToken(ctx, req.Token)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	if err := h.repo.VerifyUser(ctx, id); err != nil {
		return nil, err
	}

	log.Printf("[AUTH] Email verified for user %s", userID)

	return &authpb.VerifyEmailRes{Verified: true}, nil
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
