package hashing

import (
	"crypto/sha256"
	"fmt"

	"github.com/matthewhartstonge/argon2"
)

func HashPassword(password string) (string, error) {
	argon := argon2.DefaultConfig()

	encoded, err := argon.HashEncoded([]byte(password))
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func VerifyPassword(password, encoded []byte) (bool, error) {
	return argon2.VerifyEncoded(password, encoded)
}

func Hash(data string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
}
