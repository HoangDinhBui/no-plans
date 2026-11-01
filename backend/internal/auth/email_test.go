package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmailGateUsesAuthoritativeProviderIdentity(t *testing.T) {
	const id = "00000000-0000-4000-8000-000000000001"
	for _, tc := range []struct {
		name, body string
		status     int
		want       error
	}{
		{"verified", `{"id":"` + id + `","email":"hello@example.com","email_confirmed_at":"2026-01-01T00:00:00Z"}`, 200, nil},
		{"unverified", `{"id":"` + id + `","email":"hello@example.com","user_metadata":{"email_verified":true}}`, 200, ErrEmailUnverified},
		{"different user", `{"id":"00000000-0000-4000-8000-000000000002","email":"hello@example.com","email_confirmed_at":"2026-01-01T00:00:00Z"}`, 200, ErrUnauthorized},
		{"expired session", `{}`, 401, ErrUnauthorized},
		{"provider down", `{}`, 503, ErrAuthUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/auth/v1/user" || r.Header.Get("Authorization") != "Bearer session" || r.Header.Get("apikey") != "public-key" {
					t.Error("invalid provider request")
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			err := NewEmailGate(server.URL+"/auth/v1", "public-key", server.Client()).Check(context.Background(), "session", id)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
}

func TestEmailGateNeverForwardsTokenToRedirect(t *testing.T) {
	received := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { received = true; w.WriteHeader(200) }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer server.Close()
	err := NewEmailGate(server.URL, "key", server.Client()).Check(context.Background(), "session", "user")
	if err == nil || received {
		t.Fatal("provider redirect followed with token")
	}
}
