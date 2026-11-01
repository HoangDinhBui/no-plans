package auth

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrUnauthorized = errors.New("unauthorized")
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Principal struct{ UserID string }
type claims struct {
	jwt.RegisteredClaims
	Role      string `json:"role"`
	Anonymous bool   `json:"is_anonymous"`
}
type signingKey struct {
	key any
	alg string
}
type Verifier struct {
	issuer, audience string
	client           *http.Client
	mu               sync.RWMutex
	refreshDone      chan struct{}
	keys             map[string]signingKey
	expires          time.Time
	lastFetch        time.Time
}

func NewVerifier(issuer, audience string, client *http.Client) *Verifier {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &Verifier{issuer: strings.TrimRight(issuer, "/"), audience: audience, client: client, keys: map[string]signingKey{}}
}

func (v *Verifier) Verify(ctx context.Context, raw string) (Principal, error) {
	if len(raw) == 0 || len(raw) > 16384 {
		return Principal{}, ErrUnauthorized
	}
	c := &claims{}
	token, err := jwt.ParseWithClaims(raw, c, func(t *jwt.Token) (any, error) {
		kid, ok := t.Header["kid"].(string)
		if !ok || kid == "" || len(kid) > 256 {
			return nil, ErrUnauthorized
		}
		key, err := v.key(ctx, kid)
		if err != nil || key.alg != t.Method.Alg() {
			return nil, ErrUnauthorized
		}
		return key.key, nil
	}, jwt.WithValidMethods([]string{"ES256", "RS256"}), jwt.WithIssuer(v.issuer), jwt.WithAudience(v.audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || token == nil || !token.Valid || !uuidPattern.MatchString(c.Subject) || c.Role != "authenticated" || c.Anonymous {
		return Principal{}, ErrUnauthorized
	}
	return Principal{UserID: strings.ToLower(c.Subject)}, nil
}
