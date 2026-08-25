package lib

import (
	"crypto/rand"
	"encoding/hex"
)

func NewVerifyToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
