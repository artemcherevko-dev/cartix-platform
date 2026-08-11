package app

import (
	"context"
	"net/http"
	authpb "proto/auth"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/metadata"
)

type Handler struct {
	authClient authpb.AuthClient
}

func NewHandler(authClient authpb.AuthClient) *Handler {
	return &Handler{
		authClient: authClient,
	}
}

func (h *Handler) Register(c *gin.Context) {
	var req struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		Phone     string `json:"phone"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := contextWithMetaData(c)

	tokens, err := h.authClient.Register(ctx, &authpb.RegisterReq{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Password:  req.Password,
		Phone:     req.Phone,
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
