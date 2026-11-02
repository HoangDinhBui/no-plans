package config

import (
	"encoding/base64"
	"testing"
)

func TestLoad(t *testing.T) {
	valid := map[string]string{"DATABASE_URL": "postgres://local:local@localhost:5432/no_plans", "AUTH_ISSUER": "http://localhost:54321/auth/v1", "AUTH_AUDIENCE": "authenticated"}
	get := func(k string) string { return valid[k] }
	c, err := Load(get)
	if err != nil || c.HTTPAddr != ":8080" {
		t.Fatalf("defaults: %+v %v", c, err)
	}
	for _, key := range []string{"DATABASE_URL", "AUTH_ISSUER", "AUTH_AUDIENCE"} {
		t.Run(key, func(t *testing.T) {
			old := valid[key]
			delete(valid, key)
			defer func() { valid[key] = old }()
			if _, err := Load(get); err == nil {
				t.Fatal("missing required config accepted")
			}
		})
	}
	valid["AUTH_ISSUER"] = "not-a-url"
	if _, err := Load(get); err == nil {
		t.Fatal("invalid issuer accepted")
	}
	valid["AUTH_ISSUER"] = "https://auth.example/auth/v1"
	valid["APP_ENV"] = "production"
	valid["AUTH_API_KEY"] = "public-key"
	valid["INVITE_TOKEN_SECRET"] = base64.StdEncoding.EncodeToString(make([]byte, 32))
	valid["DATABASE_URL"] = "postgres://local:local@localhost/no_plans?sslmode=disable"
	if _, err := Load(get); err == nil {
		t.Fatal("insecure production database accepted")
	}
}

func TestProductionRequiresAuthAPIKey(t *testing.T) {
	values := map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://api@database.example/no_plans?sslmode=verify-full", "AUTH_ISSUER": "https://auth.example/auth/v1", "AUTH_AUDIENCE": "authenticated", "INVITE_TOKEN_SECRET": base64.StdEncoding.EncodeToString(make([]byte, 32))}
	get := func(key string) string { return values[key] }
	if _, err := Load(get); err == nil {
		t.Fatal("production started without provider API key")
	}
	values["AUTH_API_KEY"] = "public-key"
	if _, err := Load(get); err != nil {
		t.Fatal(err)
	}
}

func TestCORSRejectsWildcardAndNonOriginURLs(t *testing.T) {
	values := map[string]string{"DATABASE_URL": "postgres://api@localhost/no_plans", "AUTH_ISSUER": "http://localhost:54321/auth/v1", "AUTH_AUDIENCE": "authenticated"}
	get := func(k string) string { return values[k] }
	for _, value := range []string{"*", "https://example.com/path", "https://user:pass@example.com", "https://example.com?secret=x"} {
		values["CORS_ORIGINS"] = value
		if _, err := Load(get); err == nil {
			t.Fatalf("invalid origin accepted: %s", value)
		}
	}
	values["CORS_ORIGINS"] = "http://localhost:8081, https://app.example.com"
	if _, err := Load(get); err != nil {
		t.Fatal(err)
	}
}

func TestInviteSecretConfiguration(t *testing.T) {
	values := map[string]string{"DATABASE_URL": "postgres://api@localhost/no_plans", "AUTH_ISSUER": "http://localhost:54321/auth/v1", "AUTH_AUDIENCE": "authenticated"}
	get := func(k string) string { return values[k] }
	for _, secret := range []string{"not-base64", "dGVzdA=="} {
		values["INVITE_TOKEN_SECRET"] = secret
		if _, err := Load(get); err == nil {
			t.Fatal("invalid invite secret accepted")
		}
	}
	values["INVITE_TOKEN_SECRET"] = base64.StdEncoding.EncodeToString(make([]byte, 32))
	if _, err := Load(get); err != nil {
		t.Fatal(err)
	}
	values["APP_ENV"] = "production"
	values["AUTH_ISSUER"] = "https://auth.example/auth/v1"
	values["AUTH_API_KEY"] = "public-key"
	values["DATABASE_URL"] = "postgres://api@db.example/no_plans?sslmode=verify-full"
	delete(values, "INVITE_TOKEN_SECRET")
	if _, err := Load(get); err == nil {
		t.Fatal("production without invite signing secret accepted")
	}
}
