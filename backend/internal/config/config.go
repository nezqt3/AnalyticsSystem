package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DatabaseURL        string
	SQLitePath         string
	RawRetentionDays   int
	AggRetentionMonths int
	HeatmapBucketPct   int
	EventsLimit        int
	AdminEmail         string
	AdminPassword      string
	SessionSecret      string
}

func Load() Config {
	loadDotEnv()

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	sqlitePath := strings.TrimSpace(os.Getenv("SQLITE_PATH"))
	if sqlitePath == "" {
		sqlitePath = "./data/analytics.db"
	}

	return Config{
		Port:               port,
		DatabaseURL:        strings.TrimSpace(os.Getenv("DATABASE_URL")),
		SQLitePath:         sqlitePath,
		RawRetentionDays:   envInt("RAW_RETENTION_DAYS", 30),
		AggRetentionMonths: envInt("AGG_RETENTION_MONTHS", 12),
		HeatmapBucketPct:   envInt("HEATMAP_BUCKET_PCT", 5),
		EventsLimit:        envInt("EVENTS_LIMIT", 200),
		AdminEmail:         strings.TrimSpace(os.Getenv("ADMIN_EMAIL")),
		AdminPassword:      os.Getenv("ADMIN_PASSWORD"),
		SessionSecret:      strings.TrimSpace(os.Getenv("SESSION_SECRET")),
	}
}

func loadDotEnv() {
	candidates := []string{
		".env",
		filepath.Join("..", ".env"),
		filepath.Join("..", "..", ".env"),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			_ = godotenv.Overload(candidate)
			return
		}
	}
}

func envInt(key string, def int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return def
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return def
	}

	return parsed
}
