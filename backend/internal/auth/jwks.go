package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"time"
)

func (v *Verifier) cached(kid string) (signingKey, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	key, ok := v.keys[kid]
	return key, ok && time.Now().Before(v.expires)
}

func (v *Verifier) key(ctx context.Context, kid string) (signingKey, error) {
	if key, ok := v.cached(kid); ok {
		return key, nil
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	v.mu.Lock()
	if key, ok := v.keys[kid]; ok && time.Now().Before(v.expires) {
		v.mu.Unlock()
		return key, nil
	}
	if done := v.refreshDone; done != nil {
		v.mu.Unlock()
		select {
		case <-done:
			if key, ok := v.cached(kid); ok {
				return key, nil
			}
			return signingKey{}, ErrUnauthorized
		case <-ctx.Done():
			return signingKey{}, ErrUnauthorized
		}
	}
	if !v.lastFetch.IsZero() && time.Since(v.lastFetch) < time.Second {
		v.mu.Unlock()
		return signingKey{}, ErrUnauthorized
	}
	first := len(v.keys) == 0
	done := make(chan struct{})
	v.refreshDone = done
	v.mu.Unlock()
	// Only the refresh owner performs I/O; neither readers nor refresh waiters hold
	// the cache mutex. Waiters can cancel without creating another HTTP request.
	keys, err := v.fetchKeys(ctx)
	v.mu.Lock()
	if err == nil {
		v.keys = keys
		v.expires = time.Now().Add(5 * time.Minute)
	}
	key, found := v.keys[kid]
	success := err == nil && found
	if !first || !success {
		v.lastFetch = time.Now()
	}
	v.refreshDone = nil
	close(done)
	v.mu.Unlock()
	if !success {
		return signingKey{}, ErrUnauthorized
	}
	return key, nil
}

func (v *Verifier) fetchKeys(ctx context.Context) (map[string]signingKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.issuer+"/.well-known/jwks.json", nil)
	if err != nil {
		return nil, ErrUnauthorized
	}
	response, err := v.client.Do(req)
	if err != nil {
		return nil, ErrUnauthorized
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, ErrUnauthorized
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(data) > 1<<20 {
		return nil, ErrUnauthorized
	}
	var document struct {
		Keys []struct{ Kty, Kid, Alg, Use, Crv, X, Y, N, E string }
	}
	if json.Unmarshal(data, &document) != nil || len(document.Keys) > 100 {
		return nil, ErrUnauthorized
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
			return nil, ErrUnauthorized
		}
		keys[jwk.Kid] = signingKey{key: key, alg: jwk.Alg}
	}
	return keys, nil
}
