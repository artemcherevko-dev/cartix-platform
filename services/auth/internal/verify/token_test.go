package verify

import (
	"encoding/hex"
	"testing"
)

func TestNewVerifyToken(t *testing.T) {
	a, err := NewVerifyToken()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if len(a) != 64 {
		t.Errorf("want 64 hex chars, got %d", len(a))
	}

	if _, err := hex.DecodeString(a); err != nil {
		t.Errorf("token is not valid hex: %v", err)
	}

	b, err := NewVerifyToken()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if a == b {
		t.Error("tokens must be unique")
	}
}
