package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr     string
	LogLevel     slog.Level
	DBURL        string
	CORSOrigins  []string

	OIDCIssuer       string
	OIDCDiscoveryURL string
	OIDCAudience     string
	OIDCRoleClaim    string

	AllowTestReset bool
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:         env("LEVITATE_HTTP_ADDR", ":8080"),
		DBURL:            env("LEVITATE_DB_URL", ""),
		CORSOrigins:      splitCSV(env("LEVITATE_CORS_ORIGINS", "")),
		OIDCIssuer:       env("LEVITATE_OIDC_ISSUER", ""),
		OIDCDiscoveryURL: env("LEVITATE_OIDC_DISCOVERY_URL", ""),
		OIDCAudience:     env("LEVITATE_OIDC_AUDIENCE", ""),
		OIDCRoleClaim:    env("LEVITATE_OIDC_ROLE_CLAIM", "roles"),
		AllowTestReset:   strings.EqualFold(env("LEVITATE_ALLOW_TEST_RESET", "false"), "true"),
	}
	// Default discovery URL to issuer when not separately set.
	if cfg.OIDCDiscoveryURL == "" {
		cfg.OIDCDiscoveryURL = cfg.OIDCIssuer
	}

	lvl, err := parseLevel(env("LEVITATE_LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = lvl

	return cfg, nil
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	}
	return 0, fmt.Errorf("invalid log level: %q", s)
}
