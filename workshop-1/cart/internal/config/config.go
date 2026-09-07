package config

import "os"

const defaultLOMSURL = "http://localhost:8083"

type Config struct {
	LOMSURL string
}

func Load() Config {
	return Config{LOMSURL: envOr("LOMS_URL", defaultLOMSURL)}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
