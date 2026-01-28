package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	BotToken string
	AdminID  int64
	DB       DBConfig
	WebPort  string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func (d DBConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

func Load() (*Config, error) {
	adminID, err := strconv.ParseInt(getEnv("ADMIN_ID", "0"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid ADMIN_ID: %w", err)
	}

	cfg := &Config{
		BotToken: getEnv("BOT_TOKEN", ""),
		AdminID:  adminID,
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "quiz"),
			Password: getEnv("DB_PASSWORD", ""),
			Name:     getEnv("DB_NAME", "photo_quiz"),
		},
		WebPort: getEnv("WEB_PORT", "8080"),
	}

	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}