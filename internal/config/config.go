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
	ForeignEnabled         bool
	AlphaVantageKey        string
	TwelveDataKey          string
	SMTPHost               string
	SMPTPort               int
	SMPTUser               string
	SMPTPassword           string
	DefaultCurrency        string
}

func Load() *Config {
	accessExp, _ := strconv.Atoi(getEnv("ACCESS_TOKEN_EXPIRATION_MINUTES", "15"))
	refreshExp, _ := strconv.Atoi(getEnv("REFRESH_TOKEN_EXPIRATION_DAYS", "30"))
	smptPort, _ := strconv.Atoi(getEnv("SMPT_PORT", "587"))

	return &Config{
		Port:                   getEnv("PORT", "8080"),
		Env:                    getEnv("ENV", "development"),
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/fintracker?sslmode=disable"),
		JWTSecret:              getEnv("JWT_SECRET", "jwtсекретлол"),
		AccessTokenExpiration:  time.Duration(accessExp) * time.Minute,
		RefreshTokenExpiration: time.Duration(refreshExp) * 24 * time.Hour,
		MOEXEnabled:            getEnv("MOEX_ENABLED", "true") == "true",
		MOEXApiURL:             getEnv("MOEX_API_URL", "https://iss.moex.com/iss"),
		ForeignEnabled:         getEnv("FOREIGN_ENABLED", "false") == "true",
		AlphaVantageKey:        getEnv("ALPHA_VANTAGE_KEY", ""),
		TwelveDataKey:          getEnv("TWELVE_DATA_KEY", ""),
		SMTPHost:               getEnv("SMTP_HOST", ""),
		SMPTPort:               smptPort,
		SMPTUser:               getEnv("SMTP_USER", ""),
		SMPTPassword:           getEnv("SMTP_PASSWORD", ""),
		DefaultCurrency:        getEnv("DEFAULT_CURRENCY", "RUB"),
	}

}
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue

}
