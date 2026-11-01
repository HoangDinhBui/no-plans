package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
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
	refreshMu        sync.Mutex
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

func (v *Verifier) key(ctx context.Context, kid string) (signingKey, error) {
	v.mu.RLock()
	now := time.Now()
	if key, ok := v.keys[kid]; ok && now.Before(v.expires) {
		v.mu.RUnlock()
		return key, nil
	}
	v.mu.RUnlock()
	// Never block valid cached-key verification behind a network refresh.
	if !v.refreshMu.TryLock() {
		return signingKey{}, ErrUnauthorized
	}
	defer v.refreshMu.Unlock()
	v.mu.RLock()
	if key, ok := v.keys[kid]; ok && time.Now().Before(v.expires) {
		v.mu.RUnlock()
		return key, nil
	}
	// Unknown key IDs trigger one refresh, then a brief cooldown to bound abuse.
	// Permit immediate rotation after the initial fetch, which populated known keys.
	if !v.lastFetch.IsZero() && now.Sub(v.lastFetch) < time.Second {
		v.mu.RUnlock()
		return signingKey{}, ErrUnauthorized
	}
	wasEmpty := len(v.keys) == 0
	v.mu.RUnlock()
	success := false
	defer func() {
		if !wasEmpty || !success {
			v.mu.Lock()
			v.lastFetch = time.Now()
			v.mu.Unlock()
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.issuer+"/.well-known/jwks.json", nil)
	if err != nil {
		return signingKey{}, ErrUnauthorized
	}
	response, err := v.client.Do(req)
	if err != nil {
		return signingKey{}, ErrUnauthorized
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return signingKey{}, ErrUnauthorized
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return signingKey{}, ErrUnauthorized
	}
	var document struct {
		Keys []struct{ Kty, Kid, Alg, Use, Crv, X, Y, N, E string }
	}
	if json.Unmarshal(data, &document) != nil || len(document.Keys) > 100 {
		return signingKey{}, ErrUnauthorized
	}
	keys := map[string]signingKey{}
	for _, jwk := range document.Keys {
		if jwk.Kid == "" || (jwk.Use != "" && jwk.Use != "sig") {
			continue
		}
		var key any
		switch {
		case jwk.Kty == "EC" && jwk.Crv == "P-256" && jwk.Alg == "ES256":
			x, e1 := base64.RawURLEncoding.DecodeString(jwk.X)
			y, e2 := base64.RawURLEncoding.DecodeString(jwk.Y)
			if e1 != nil || e2 != nil || len(x) != 32 || len(y) != 32 {
				continue
			}
			public := &ecdsa.PublicKey{Curve: elliptic.P256(), X: new(big.Int).SetBytes(x), Y: new(big.Int).SetBytes(y)}
			if !public.Curve.IsOnCurve(public.X, public.Y) {
				continue
			}
			key = public
		case jwk.Kty == "RSA" && jwk.Alg == "RS256":
			n, e1 := base64.RawURLEncoding.DecodeString(jwk.N)
			e, e2 := base64.RawURLEncoding.DecodeString(jwk.E)
			if e1 != nil || e2 != nil || len(n) < 256 || len(n) > 1024 || len(e) == 0 || len(e) > 4 {
				continue
			}
			exponent := new(big.Int).SetBytes(e).Int64()
			if exponent < 3 || exponent > 1<<31-1 || exponent%2 == 0 {
				continue
			}
			key = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(exponent)}
		default:
			continue
		}
		if _, duplicate := keys[jwk.Kid]; duplicate {
			return signingKey{}, ErrUnauthorized
		}
		keys[jwk.Kid] = signingKey{key: key, alg: jwk.Alg}
	}
	v.mu.Lock()
	v.keys = keys
	v.expires = time.Now().Add(5 * time.Minute)
	v.mu.Unlock()
	key, ok := keys[kid]
	if !ok {
		return signingKey{}, ErrUnauthorized
	}
	success = true
	return key, nil
}
