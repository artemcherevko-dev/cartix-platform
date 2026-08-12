package jwt

import (
	"auth/internal/app/db"
	"crypto/rand"
	"encoding/base64"
	"os"
	"shared"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateTokenPair(userID string, role db.UserRole) (string, string, error) {
	secret := os.Getenv("JWT_SECRET")
	now := time.Now()

	// Access token
	claims := shared.Claims{
		Role: string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Issuer:    "cartix-platform",
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(15 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	access, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}

	// Refresh token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	refresh := base64.RawURLEncoding.EncodeToString(b)

	// Access Refresh Err
	return access, refresh, nil
}
