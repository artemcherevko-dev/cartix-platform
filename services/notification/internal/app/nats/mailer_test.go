package nats

import (
	"strings"
	"testing"
)

func TestWelcomeTemplateRenders(t *testing.T) {
	link := "http://localhost:3000/v1/auth/verify?token=abc123"

	out, err := renderWelcome(UserTemplateData{
		FirstName:        "John",
		LastName:         "Doe",
		VerificationLink: link,
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	for _, want := range []string{"John", "Doe", link, "Verify my email", "15 minutes"} {
		if !strings.Contains(out, want) {
			t.Errorf("template output missing %q", want)
		}
	}

	if strings.Contains(out, "{{") {
		t.Error("template contains unrendered placeholders")
	}
}

func TestVerificationLink(t *testing.T) {
	m := NewMailer(nil, "http://localhost:3000")

	got := m.VerificationLink("tok123")
	want := "http://localhost:3000/v1/auth/verify?token=tok123"

	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
