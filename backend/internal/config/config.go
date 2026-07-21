// Package config loads server configuration from the environment.
package config

import "os"

type Config struct {
	DatabaseURL   string
	Port          string
	SessionSecret string
	CookieSecure  bool // Secure flag on the session cookie; off for local http dev.
}

// Load reads configuration from the environment, falling back to local dev defaults.
func Load() Config {
	return Config{
		DatabaseURL:   env("DATABASE_URL", "postgres://focus:focus@localhost:5432/focus?sslmode=disable"),
		Port:          env("PORT", "8080"),
		SessionSecret: env("SESSION_SECRET", "dev-insecure-change-me"),
		CookieSecure:  env("COOKIE_SECURE", "") == "true",
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
