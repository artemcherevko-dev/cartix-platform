package app

import (
	"context"
	"net/http"
	authpb "proto/auth"
	profilepb "proto/profile"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
)

type Handler struct {
	authClient    authpb.AuthClient
	profileClient profilepb.ProfileClient
}

func NewHandler(authClient authpb.AuthClient, profileClient profilepb.ProfileClient) *Handler {
	return &Handler{
		authClient:    authClient,
		profileClient: profileClient,
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		Phone       string `json:"phone"`
		FirstName   string `json:"first_name"`
		MiddleName  string `json:"middle_name"`
		LastName    string `json:"last_name"`
		DateOfBirth string `json:"date_of_birth"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := contextWithMetaData(c)

	tokens, err := h.authClient.Register(ctx, &authpb.RegisterReq{
		FirstName:   req.FirstName,
		MiddleName:  req.MiddleName,
		LastName:    req.LastName,
		Email:       req.Email,
		Password:    req.Password,
		Phone:       req.Phone,
		DateOfBirth: req.DateOfBirth,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tokens == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tokens is nil"})
		return
	}

	setCookie(c, tokens.AccessToken, tokens.RefreshToken)

	c.JSON(200, gin.H{"status": "registered"})
}

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := contextWithMetaData(c)

	tokens, err := h.authClient.Login(ctx, &authpb.LoginReq{
		Login:    req.Login,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tokens == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tokens is nil"})
		return
	}

	setCookie(c, tokens.AccessToken, tokens.RefreshToken)
	c.JSON(200, gin.H{"status": "logged in"})
}

func (h *Handler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx := contextWithMetaData(c)

	tokens, err := h.authClient.Refresh(ctx, &authpb.RefreshReq{
		RefreshToken: refreshToken,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tokens == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tokens is nil"})
		return
	}

	setCookie(c, tokens.AccessToken, tokens.RefreshToken)
	c.JSON(200, gin.H{"status": "refreshed"})
}

func (h *Handler) ValidateToken(c *gin.Context) {
	accessToken, err := c.Cookie("access_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authClient.Validate(c.Request.Context(), &authpb.ValidateTokenReq{
		AccessToken: accessToken,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(200, gin.H{"status": "validated", "user": user.User})
}

func (h *Handler) GetProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not exist"})
		return
	}

	profile, err := h.profileClient.GetProfile(c.Request.Context(), &profilepb.GetProfileReq{UserId: userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"profile": profile})
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not exist"})
		return
	}

	var input struct {
		FirstName   *string `json:"first_name"`
		MiddleName  *string `json:"middle_name"`
		LastName    *string `json:"last_name"`
		DateOfBirth *string `json:"date_of_birth"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.profileClient.UpdateProfile(c.Request.Context(), &profilepb.UpdateProfileReq{
		UserId:      userID,
		FirstName:   input.FirstName,
		MiddleName:  input.MiddleName,
		LastName:    input.LastName,
		DateOfBirth: input.DateOfBirth,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"profile": profile})
}

func contextWithMetaData(c *gin.Context) context.Context {
	userAgent := c.GetHeader("User-Agent")
	clientIP := c.ClientIP()

	md := metadata.Pairs(
		"x-client-ip", clientIP,
		"x-user-agent", userAgent,
	)

	ctx := metadata.NewOutgoingContext(c.Request.Context(), md)
	return ctx
}

func setCookie(c *gin.Context, access, refresh string) {
	c.SetCookie("access_token", access, 60*60*24*7, "/", "", false, true)
	c.SetCookie("refresh_token", refresh, 60*60*24*7, "/", "", false, true)
}
