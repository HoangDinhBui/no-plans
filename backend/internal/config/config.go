package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
)

type Config struct {
	HTTPAddr     string
	DatabaseURL  string
	AuthIssuer   string
	AuthAudience string
	AuthAPIKey   string
	CORSOrigins  []string
	InviteSecret []byte
	Environment  string
}

// Load validates configuration without including secret values in errors.
func Load(getenv func(string) string) (Config, error) {
	c := Config{HTTPAddr: getenv("HTTP_ADDR"), DatabaseURL: getenv("DATABASE_URL"), AuthIssuer: getenv("AUTH_ISSUER"), AuthAudience: getenv("AUTH_AUDIENCE"), Environment: getenv("APP_ENV")}
	c.AuthAPIKey = getenv("AUTH_API_KEY")
	if raw := strings.TrimSpace(getenv("INVITE_TOKEN_SECRET")); raw != "" {
		secret, decodeErr := base64.StdEncoding.DecodeString(raw)
		if decodeErr != nil || len(secret) != 32 {
			return Config{}, fmt.Errorf("INVITE_TOKEN_SECRET must be base64 of 32 random bytes")
		}
		c.InviteSecret = secret
	}
	if c.HTTPAddr == "" {
		c.HTTPAddr = ":8080"
	}
	if c.Environment == "" {
		c.Environment = "development"
	}
	if c.Environment != "development" && c.Environment != "test" && c.Environment != "production" {
		return Config{}, fmt.Errorf("APP_ENV must be development, test or production")
	}
	for key, value := range map[string]string{"DATABASE_URL": c.DatabaseURL, "AUTH_ISSUER": c.AuthIssuer, "AUTH_AUDIENCE": c.AuthAudience} {
		if strings.TrimSpace(value) == "" {
			return Config{}, fmt.Errorf("%s is required", key)
		}
	}
	db, err := url.Parse(c.DatabaseURL)
	if err != nil || db == nil || (db.Scheme != "postgres" && db.Scheme != "postgresql") || db.Host == "" {
		return Config{}, fmt.Errorf("DATABASE_URL must be a PostgreSQL URL")
	}
	issuer, err := url.Parse(c.AuthIssuer)
	if err != nil || issuer == nil || (issuer.Scheme != "http" && issuer.Scheme != "https") || issuer.Host == "" || issuer.RawQuery != "" || issuer.Fragment != "" {
		return Config{}, fmt.Errorf("AUTH_ISSUER must be an HTTP(S) issuer URL")
	}
	if c.Environment == "production" {
		if len(c.InviteSecret) != 32 {
			return Config{}, fmt.Errorf("production INVITE_TOKEN_SECRET is required")
		}
		if strings.TrimSpace(c.AuthAPIKey) == "" {
			return Config{}, fmt.Errorf("production AUTH_API_KEY is required")
		}
		if issuer.Scheme != "https" {
			return Config{}, fmt.Errorf("production AUTH_ISSUER requires HTTPS")
		}
		if db.Query().Get("sslmode") != "verify-full" {
			return Config{}, fmt.Errorf("production DATABASE_URL requires sslmode=verify-full")
		}
	}
	if value := getenv("CORS_ORIGINS"); strings.TrimSpace(value) != "" {
		for _, candidate := range strings.Split(value, ",") {
			origin := strings.TrimSpace(candidate)
			u, parseErr := url.Parse(origin)
			if parseErr != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || strings.Contains(origin, "*") {
				return Config{}, fmt.Errorf("CORS_ORIGINS must contain exact HTTP(S) origins")
			}
			if c.Environment == "production" && u.Scheme != "https" {
				return Config{}, fmt.Errorf("production CORS_ORIGINS requires HTTPS")
			}
			c.CORSOrigins = append(c.CORSOrigins, origin)
		}
	}
	c.AuthIssuer = strings.TrimRight(c.AuthIssuer, "/")
	return c, nil
}
