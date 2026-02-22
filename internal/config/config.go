package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                   string
	Env                    string
	DatabaseURL            string
	JWTSecret              string
	AccessTokenExpiration  time.Duration
	RefreshTokenExpiration time.Duration
	MOEXEnabled            bool
	MOEXApiURL             string
	DefaultCurrency        string

	OllamaURL   string
	OllamaModel string
}

func Load() *Config {
	accessExp, _ := strconv.Atoi(getEnv("ACCESS_TOKEN_EXPIRATION_MINUTES", "15"))
	refreshExp, _ := strconv.Atoi(getEnv("REFRESH_TOKEN_EXPIRATION_DAYS", "30"))

	return &Config{
		Port:                   getEnv("PORT", "8080"),
		Env:                    getEnv("ENV", "development"),
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/fintracker?sslmode=disable"),
		JWTSecret:              getEnv("JWT_SECRET", "jwtсекретлол"),
		AccessTokenExpiration:  time.Duration(accessExp) * time.Minute,
		RefreshTokenExpiration: time.Duration(refreshExp) * 24 * time.Hour,
		MOEXEnabled:            getEnv("MOEX_ENABLED", "true") == "true",
		MOEXApiURL:             getEnv("MOEX_API_URL", "https://iss.moex.com/iss"),
		DefaultCurrency:        getEnv("DEFAULT_CURRENCY", "RUB"),

		OllamaURL:   getEnv("OLLAMA_URL", "http://localhost:11434"),
		OllamaModel: getEnv("OLLAMA_MODEL", "llama3.2:3b"),
	}

}
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue

}
