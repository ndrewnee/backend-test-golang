package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	DatabaseURL      string
	HTTPAddr         string
	SkinportBaseURL  string
	SkinportCacheTTL time.Duration
	SkinportTimeout  time.Duration
	RunMigrations    bool
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/backend_test?sslmode=disable"),
		HTTPAddr:         getEnv("HTTP_ADDR", ":8080"),
		SkinportBaseURL:  getEnv("SKINPORT_BASE_URL", "https://api.skinport.com/v1"),
		SkinportCacheTTL: 5 * time.Minute,
		SkinportTimeout:  10 * time.Second,
		RunMigrations:    true,
	}

	var err error
	if raw := os.Getenv("SKINPORT_CACHE_TTL"); raw != "" {
		cfg.SkinportCacheTTL, err = time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse SKINPORT_CACHE_TTL: %w", err)
		}
	}

	if raw := os.Getenv("SKINPORT_TIMEOUT"); raw != "" {
		cfg.SkinportTimeout, err = time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse SKINPORT_TIMEOUT: %w", err)
		}
	}

	if raw := os.Getenv("RUN_MIGRATIONS"); raw != "" {
		cfg.RunMigrations, err = strconv.ParseBool(raw)
		if err != nil {
			return Config{}, fmt.Errorf("parse RUN_MIGRATIONS: %w", err)
		}
	}

	if cfg.SkinportCacheTTL <= 0 {
		return Config{}, fmt.Errorf("SKINPORT_CACHE_TTL must be positive")
	}
	if cfg.SkinportTimeout <= 0 {
		return Config{}, fmt.Errorf("SKINPORT_TIMEOUT must be positive")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
