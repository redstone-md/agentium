package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr      string
	ChromeBin     string
	DefaultWidth  int
	DefaultHeight int
}

func Load() Config {
	return Config{
		HTTPAddr:      env("AGENTIUM_HTTP_ADDR", ":8080"),
		ChromeBin:     os.Getenv("AGENTIUM_CHROME_BIN"),
		DefaultWidth:  envInt("AGENTIUM_VIEWPORT_WIDTH", 1280),
		DefaultHeight: envInt("AGENTIUM_VIEWPORT_HEIGHT", 800),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
