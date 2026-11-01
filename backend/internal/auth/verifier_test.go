package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestVerifyJWTAndRotation(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	second, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	var mu sync.Mutex
	current := key
	kid := "first"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/v1/.well-known/jwks.json" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		mu.Lock()
		defer mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kty": "EC", "crv": "P-256", "alg": "ES256", "use": "sig", "kid": kid, "x": base64.RawURLEncoding.EncodeToString(current.X.FillBytes(make([]byte, 32))), "y": base64.RawURLEncoding.EncodeToString(current.Y.FillBytes(make([]byte, 32)))}}})
	}))
	defer server.Close()
	issuer := server.URL + "/auth/v1"
	v := NewVerifier(issuer, "authenticated", server.Client())
	claims := func() jwt.MapClaims {
		return jwt.MapClaims{"sub": "00000000-0000-4000-8000-000000000001", "iss": issuer, "aud": "authenticated", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "role": "authenticated", "is_anonymous": false}
	}
	sign := func(c jwt.MapClaims, k *ecdsa.PrivateKey, id string) string {
		t.Helper()
		token := jwt.NewWithClaims(jwt.SigningMethodES256, c)
		token.Header["kid"] = id
		raw, err := token.SignedString(k)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	principal, err := v.Verify(context.Background(), sign(claims(), key, "first"))
	if err != nil || principal.UserID != "00000000-0000-4000-8000-000000000001" {
		t.Fatalf("%+v %v", principal, err)
	}
	for _, tc := range []struct {
		name, field string
		value       any
	}{
		{"issuer", "iss", "https://attacker.invalid"}, {"audience", "aud", "service_role"}, {"expiry", "exp", time.Now().Add(-time.Hour).Unix()},
		{"missing expiry", "exp", nil}, {"subject", "sub", "not-a-uuid"}, {"service role", "role", "service_role"}, {"anonymous", "is_anonymous", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := claims()
			c[tc.field] = tc.value
			if _, err := v.Verify(context.Background(), sign(c, key, "first")); err == nil {
				t.Fatal("invalid identity accepted")
			}
		})
	}
	if _, err := v.Verify(context.Background(), sign(claims(), second, "first")); err == nil {
		t.Fatal("forged signature accepted")
	}
	mu.Lock()
	current = second
	kid = "second"
	mu.Unlock()
	if _, err := v.Verify(context.Background(), sign(claims(), second, "second")); err != nil {
		t.Fatal("new signing key not discovered", err)
	}
	hmac := jwt.NewWithClaims(jwt.SigningMethodHS256, claims())
	hmac.Header["kid"] = "first"
	raw, _ := hmac.SignedString([]byte("bad"))
	if _, err := v.Verify(context.Background(), raw); err == nil {
		t.Fatal("algorithm confusion accepted")
	}
}

func TestJWKSFailureIsClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "unavailable", 503) }))
	defer server.Close()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"sub": "00000000-0000-4000-8000-000000000001", "iss": server.URL, "aud": "authenticated", "exp": time.Now().Add(time.Hour).Unix(), "role": "authenticated"})
	token.Header["kid"] = "missing"
	raw, _ := token.SignedString(key)
	if _, err := NewVerifier(server.URL, "authenticated", server.Client()).Verify(context.Background(), raw); err == nil {
		t.Fatal("accepted token without trusted key")
	}
}

func TestCachedKeyDoesNotWaitForSlowRefresh(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(entered) })
		<-release
		http.Error(w, "unavailable", 503)
	}))
	defer server.Close()
	defer close(release)
	v := NewVerifier(server.URL, "authenticated", server.Client())
	v.keys["known"] = signingKey{key: &key.PublicKey, alg: "ES256"}
	v.expires = time.Now().Add(time.Minute)
	makeToken := func(kid string) string {
		token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"sub": "00000000-0000-4000-8000-000000000001", "iss": server.URL, "aud": "authenticated", "role": "authenticated", "exp": time.Now().Add(time.Hour).Unix()})
		token.Header["kid"] = kid
		raw, _ := token.SignedString(key)
		return raw
	}
	unknownDone := make(chan struct{})
	go func() { defer close(unknownDone); _, _ = v.Verify(context.Background(), makeToken("unknown")) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("refresh never started")
	}
	result := make(chan error, 1)
	go func() { _, err := v.Verify(context.Background(), makeToken("known")); result <- err }()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("cached-key verification blocked on JWKS refresh")
	}
}
