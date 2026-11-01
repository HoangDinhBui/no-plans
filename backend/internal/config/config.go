package config

import (
	"fmt"
	"net/url"
	"strings"
)

type Config struct {
	HTTPAddr     string
	DatabaseURL  string
	AuthIssuer   string
	AuthAudience string
	Environment  string
}

// Load validates configuration without including secret values in errors.
func Load(getenv func(string) string) (Config, error) {
	c := Config{HTTPAddr: getenv("HTTP_ADDR"), DatabaseURL: getenv("DATABASE_URL"), AuthIssuer: getenv("AUTH_ISSUER"), AuthAudience: getenv("AUTH_AUDIENCE"), Environment: getenv("APP_ENV")}
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
		if issuer.Scheme != "https" {
			return Config{}, fmt.Errorf("production AUTH_ISSUER requires HTTPS")
		}
		if db.Query().Get("sslmode") != "verify-full" {
			return Config{}, fmt.Errorf("production DATABASE_URL requires sslmode=verify-full")
		}
	}
	c.AuthIssuer = strings.TrimRight(c.AuthIssuer, "/")
	return c, nil
}
