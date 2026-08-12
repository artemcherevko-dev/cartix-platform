package jwt

import (
	"os"
	"shared"

	"github.com/golang-jwt/jwt/v5"
)

func ParseAccessToken(access string) (*shared.Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	token, err := jwt.ParseWithClaims(access, &shared.Claims{}, func(token *jwt.Token) (any, error) { return []byte(secret), nil })

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*shared.Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil

}
