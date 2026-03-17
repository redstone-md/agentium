package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr      string
	ChromeBin     string
	Headless      bool
	DefaultWidth  int
	DefaultHeight int
	UseLeakless   bool
	GeoIPEndpoint string
	GeoIPTimeout  time.Duration
}

func Load() Config {
	return Config{
		HTTPAddr:      env("AGENTIUM_HTTP_ADDR", ":8080"),
		ChromeBin:     os.Getenv("AGENTIUM_CHROME_BIN"),
		Headless:      envBool("AGENTIUM_HEADLESS", false),
		DefaultWidth:  envInt("AGENTIUM_VIEWPORT_WIDTH", 1280),
		DefaultHeight: envInt("AGENTIUM_VIEWPORT_HEIGHT", 800),
		UseLeakless:   envBool("AGENTIUM_LEAKLESS", true),
		GeoIPEndpoint: env("AGENTIUM_GEOIP_ENDPOINT", "https://ipwho.is/"),
		GeoIPTimeout:  envDurationSeconds("AGENTIUM_GEOIP_TIMEOUT_SECONDS", 8*time.Second),
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

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envDurationSeconds(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}

	return time.Duration(seconds) * time.Second
}
