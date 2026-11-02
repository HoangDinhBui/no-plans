package httpapi

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	urlpkg "net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/HoangDinhBui/no-plans/backend/internal/auth"
	"github.com/HoangDinhBui/no-plans/backend/internal/ids"
	"github.com/HoangDinhBui/no-plans/backend/internal/store"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

type appFixture struct {
	handler    http.Handler
	token      func(string) string
	unverified string
	db         *store.Store
}

func setupApp(t *testing.T) appFixture {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	root, err := pgx.Connect(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	schema := "http_test_" + strings.ReplaceAll(ids.New(), "-", "")
	if _, err = root.Exec(context.Background(), "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		root.Close(context.Background())
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = root.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		_ = root.Close(context.Background())
	})
	scoped, err := urlpkg.Parse(url)
	if err != nil {
		t.Fatal(err)
	}
	query := scoped.Query()
	query.Set("search_path", schema)
	scoped.RawQuery = query.Encode()
	db, err := store.Open(context.Background(), scoped.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err = db.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	unverified := ids.New()
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/v1/.well-known/jwks.json":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"kty": "EC", "crv": "P-256", "alg": "ES256", "kid": "test-key", "x": base64.RawURLEncoding.EncodeToString(key.X.FillBytes(make([]byte, 32))), "y": base64.RawURLEncoding.EncodeToString(key.Y.FillBytes(make([]byte, 32)))}}})
		case "/auth/v1/user":
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			c := jwt.MapClaims{}
			_, err := jwt.ParseWithClaims(raw, c, func(*jwt.Token) (any, error) { return &key.PublicKey, nil }, jwt.WithValidMethods([]string{"ES256"}))
			if err != nil {
				w.WriteHeader(401)
				return
			}
			body := map[string]any{"id": c["sub"], "email": "test@example.com"}
			if c["sub"] != unverified {
				body["email_confirmed_at"] = "2026-01-01T00:00:00Z"
			}
			_ = json.NewEncoder(w).Encode(body)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(provider.Close)
	issuer := provider.URL + "/auth/v1"
	token := func(id string) string {
		t.Helper()
		tok := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"sub": id, "role": "authenticated", "iss": issuer, "aud": "authenticated", "exp": time.Now().Add(time.Hour).Unix()})
		tok.Header["kid"] = "test-key"
		raw, err := tok.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	return appFixture{handler: NewApplicationWithOptions(db, auth.NewVerifier(issuer, "authenticated", provider.Client()), auth.NewEmailGate(issuer, "test-key", provider.Client()), Options{Origins: []string{"https://app.no-plans.test"}, InviteSecret: bytes.Repeat([]byte{7}, 32)}), token: token, unverified: unverified, db: db}
}

func call(t *testing.T, h http.Handler, method, path, token, body, key string, want int) map[string]any {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != want {
		t.Fatalf("%s %s: status=%d want=%d body=%s", method, path, w.Code, want, w.Body.String())
	}
	if w.Header().Get("X-Request-ID") == "" {
		t.Fatal("request id missing")
	}
	var result map[string]any
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func TestAuthenticatedProfilesAndGroups(t *testing.T) {
	f := setupApp(t)
	a, b, c := ids.New(), ids.New(), ids.New()
	ta, tb, tc := f.token(a), f.token(b), f.token(c)
	call(t, f.handler, "GET", "/v1/me", "", "", "", 401)
	call(t, f.handler, "GET", "/v1/me", "forged", "", "", 401)
	if p := call(t, f.handler, "GET", "/v1/me", ta, "", "", 200); p["id"] != a {
		t.Fatal("wrong actor profile")
	}
	call(t, f.handler, "GET", "/v1/me", tb, "", "", 200)
	call(t, f.handler, "GET", "/v1/me", tc, "", "", 200)
	call(t, f.handler, "PATCH", "/v1/me", ta, `{"display_name":"Đình","avatar_url":""}`, "", 200)
	call(t, f.handler, "PATCH", "/v1/me", ta, `{"display_name":"Spoof","id":"`+b+`"}`, "", 400)
	if p := call(t, f.handler, "GET", "/v1/me", ta, "", "", 200); p["display_name"] != "Đình" {
		t.Fatal("profile provisioning overwrote name")
	}
	call(t, f.handler, "POST", "/v1/groups", ta, `{"name":"Cà phê"}`, "", 400)
	g := call(t, f.handler, "POST", "/v1/groups", ta, `{"name":"Cà phê"}`, "create-group", 201)
	gid := g["id"].(string)
	replay := call(t, f.handler, "POST", "/v1/groups", ta, `{"name":"Cà phê"}`, "create-group", 201)
	if replay["id"] != gid {
		t.Fatal("HTTP retry duplicated group")
	}
	call(t, f.handler, "POST", "/v1/groups", ta, `{"name":"Changed"}`, "create-group", 409)
	call(t, f.handler, "GET", "/v1/groups/"+gid, tb, "", "", 403)
	req := call(t, f.handler, "POST", "/v1/friend-requests", ta, `{"recipient_id":"`+b+`"}`, "friend", 201)
	call(t, f.handler, "POST", "/v1/friend-requests/"+req["id"].(string)+"/accept", tb, "", "", 200)
	call(t, f.handler, "POST", "/v1/groups/"+gid+"/members/"+b, ta, "", "", 200)
	group := call(t, f.handler, "GET", "/v1/groups/"+gid, tb, "", "", 200)
	if len(group["members"].([]any)) != 2 {
		t.Fatal("group member missing")
	}
	call(t, f.handler, "POST", "/v1/groups/"+gid+"/members/"+c, tb, "", "", 403)
	call(t, f.handler, "POST", "/v1/friend-requests", f.token(f.unverified), `{"recipient_id":"`+a+`"}`, "unverified", 403)
	call(t, f.handler, "GET", "/v1/groups/not-a-uuid", ta, "", "", 400)
	call(t, f.handler, "PUT", "/v1/groups/"+gid+"/owner", ta, `{"user_id":"invalid"}`, "", 400)
	call(t, f.handler, "PATCH", "/v1/me", ta, `{"display_name":"`+strings.Repeat("x", 70000)+`"}`, "", 413)
	call(t, f.handler, "POST", "/v1/me", ta, `{}`, "", 405)
	call(t, f.handler, "GET", "/v1/unknown", ta, "", "", 404)
}
