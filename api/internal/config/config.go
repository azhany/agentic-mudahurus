// Package config loads runtime configuration from the environment.
// All MUDAHURUS services read configuration from env vars (12-factor); see .env.example.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env      string
	HTTPAddr string
	LogLevel string

	DatabaseURL string

	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessTTL     time.Duration
	JWTRefreshTTL    time.Duration

	S3Endpoint      string
	S3AccessKey     string
	S3SecretKey     string
	S3Bucket        string
	S3UseSSL        bool
	S3PublicBaseURL string

	RAGBaseURL string

	EventsSink        string
	AirflowTriggerURL string

	SMTPSink string
	SMTPFrom string
}

func Load() (*Config, error) {
	c := &Config{
		Env:               env("APP_ENV", "development"),
		HTTPAddr:          env("API_HTTP_ADDR", ":8080"),
		LogLevel:          env("API_LOG_LEVEL", "info"),
		DatabaseURL:       env("DATABASE_URL", "postgres://mudahurus:mudahurus@localhost:5432/mudahurus?sslmode=disable"),
		JWTAccessSecret:   env("JWT_ACCESS_SECRET", "dev-access-secret-change-me"),
		JWTRefreshSecret:  env("JWT_REFRESH_SECRET", "dev-refresh-secret-change-me"),
		S3Endpoint:        env("S3_ENDPOINT", "localhost:9000"),
		S3AccessKey:       env("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:       env("S3_SECRET_KEY", "minioadmin"),
		S3Bucket:          env("S3_BUCKET", "mudahurus"),
		S3UseSSL:          envBool("S3_USE_SSL", false),
		S3PublicBaseURL:   env("S3_PUBLIC_BASE_URL", "http://localhost:9000"),
		RAGBaseURL:        env("RAG_BASE_URL", "http://localhost:8000"),
		EventsSink:        env("EVENTS_SINK", "log"),
		AirflowTriggerURL: env("AIRFLOW_TRIGGER_URL", ""),
		SMTPSink:          env("SMTP_SINK", "log"),
		SMTPFrom:          env("SMTP_FROM", "no-reply@mudahurus.my"),
	}
	var err error
	if c.JWTAccessTTL, err = time.ParseDuration(env("JWT_ACCESS_TTL", "15m")); err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_TTL: %w", err)
	}
	if c.JWTRefreshTTL, err = time.ParseDuration(env("JWT_REFRESH_TTL", "168h")); err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_TTL: %w", err)
	}
	return c, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}
