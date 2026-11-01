package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrEmailUnverified = errors.New("email_unverified")
var ErrAuthUnavailable = errors.New("auth_unavailable")

type EmailGate struct {
	issuer, apiKey string
	client         *http.Client
}

func NewEmailGate(issuer, apiKey string, client *http.Client) *EmailGate {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	safe := *client
	safe.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &EmailGate{issuer: strings.TrimRight(issuer, "/"), apiKey: apiKey, client: &safe}
}

// Check reads the provider's authoritative user record. User-editable metadata
// and optional custom JWT claims cannot prove email ownership.
func (g *EmailGate) Check(ctx context.Context, token, expectedID string) error {
	if g == nil || g.apiKey == "" {
		return ErrAuthUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.issuer+"/user", nil)
	if err != nil {
		return ErrAuthUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", g.apiKey)
	response, err := g.client.Do(req)
	if err != nil {
		return ErrAuthUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == 401 || response.StatusCode == 403 {
		return ErrUnauthorized
	}
	if response.StatusCode != 200 {
		return ErrAuthUnavailable
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 65537))
	if err != nil || len(data) > 65536 {
		return ErrAuthUnavailable
	}
	var user struct {
		ID        string     `json:"id"`
		Email     string     `json:"email"`
		Confirmed *time.Time `json:"email_confirmed_at"`
		Anonymous bool       `json:"is_anonymous"`
	}
	if json.Unmarshal(data, &user) != nil {
		return ErrAuthUnavailable
	}
	if user.ID != expectedID || user.Anonymous {
		return ErrUnauthorized
	}
	if user.Email == "" || user.Confirmed == nil || user.Confirmed.IsZero() {
		return ErrEmailUnverified
	}
	return nil
}
