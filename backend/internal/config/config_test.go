package config

import "testing"

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
	valid["DATABASE_URL"] = "postgres://local:local@localhost/no_plans?sslmode=disable"
	if _, err := Load(get); err == nil {
		t.Fatal("insecure production database accepted")
	}
}
