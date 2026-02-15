package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr           string
	LogLevel           slog.Level
	LLMProvider        string
	LLMModel           string
	LLMFallbackModels  []string
	OpenRouterAPIKey   string
	OpenRouterBaseURL  string
	LLMTimeout         time.Duration
}

func Load() (Config, error) {
	c := Config{
		HTTPAddr:          envOr("HTTP_ADDR", ":8080"),
		LLMProvider:       envOr("LLM_PROVIDER", "openrouter"),
		LLMModel:          envOr("LLM_MODEL", "qwen/qwen3-4b:free"),
		OpenRouterAPIKey:  os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterBaseURL: envOr("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
		LLMFallbackModels: parseFallbackModels(os.Getenv("LLM_FALLBACK_MODELS")),
		LLMTimeout:        10 * time.Second,
	}

	if v := os.Getenv("LLM_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("invalid LLM_TIMEOUT %q: %w", v, err)
		}
		c.LLMTimeout = d
	}

	level, err := parseLogLevel(envOr("LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}
	c.LogLevel = level

	if c.LLMProvider == "openrouter" && c.OpenRouterAPIKey == "" {
		return Config{}, fmt.Errorf("OPENROUTER_API_KEY is required when LLM_PROVIDER=openrouter")
	}

	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseFallbackModels(s string) []string {
	if s == "" {
		return nil
	}
	var models []string
	for _, m := range strings.Split(s, ",") {
		m = strings.TrimSpace(m)
		if m != "" {
			models = append(models, m)
		}
	}
	return models
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid LOG_LEVEL %q", s)
	}
}
