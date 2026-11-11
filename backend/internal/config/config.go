package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppEnv, AppOrigin, Port, DatabaseURL, AssistantMode              string
	PubSubEmulatorHost, PubSubAPIEndpoint, CloudProject, PubSubTopic string
	GeminiAPIKey, GeminiModel, GeminiAPIEndpoint                     string
	FrontendDist, RateLimitMode                                      string
	DBMaxConns                                                       int32
	ReservationTTLSeconds                                            int
}

func Load() (Config, error) {
	c := Config{
		AppEnv: value("APP_ENV", "local"), AppOrigin: value("APP_ORIGIN", "http://localhost:8080"),
		Port: value("PORT", "8080"), DatabaseURL: os.Getenv("DATABASE_URL"),
		AssistantMode: value("ASSISTANT_MODE", "template"), DBMaxConns: 5,
		PubSubEmulatorHost: os.Getenv("PUBSUB_EMULATOR_HOST"), PubSubAPIEndpoint: os.Getenv("PUBSUB_API_ENDPOINT"), CloudProject: os.Getenv("GOOGLE_CLOUD_PROJECT"), PubSubTopic: value("PUBSUB_TOPIC", "flashcart-events"),
		GeminiAPIKey: os.Getenv("GEMINI_API_KEY"), GeminiModel: value("GEMINI_MODEL", "gemini-3.8-flash"), GeminiAPIEndpoint: value("GEMINI_API_ENDPOINT", "https://generativelanguage.googleapis.com"),
		FrontendDist: value("FRONTEND_DIST", "../frontend/dist"), ReservationTTLSeconds: 300,
		RateLimitMode: value("RATE_LIMIT_MODE", "normal"),
	}
	if raw := os.Getenv("DB_MAX_CONNS"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			return c, fmt.Errorf("DB_MAX_CONNS must be between 1 and 100")
		}
		c.DBMaxConns = int32(n)
	}
	if raw := os.Getenv("RESERVATION_TTL_SECONDS"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 60 || n > 900 {
			return c, fmt.Errorf("RESERVATION_TTL_SECONDS must be between 60 and 900")
		}
		c.ReservationTTLSeconds = n
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if c.AssistantMode != "template" && c.AssistantMode != "gemini" {
		return c, fmt.Errorf("ASSISTANT_MODE must be template or gemini")
	}
	if c.RateLimitMode != "normal" && c.RateLimitMode != "load" {
		return c, fmt.Errorf("RATE_LIMIT_MODE must be normal or load")
	}
	if c.AppEnv == "production" && c.RateLimitMode == "load" {
		return c, fmt.Errorf("RATE_LIMIT_MODE=load is forbidden in production")
	}
	if c.AppEnv == "production" && c.AppOrigin == "" {
		return c, fmt.Errorf("APP_ORIGIN is required in production")
	}
	return c, nil
}

func value(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
