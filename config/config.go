package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type RajaOngkirConfig struct {
	APIKey   string
	BaseURL  string
	OriginID int
}

type AppConfig struct {
	BaseURL     string
	Port        string
	JWTSecret   string
	SMTP        email.SMTPConfig
	AppEnv      string
	GoogleOAuth *oauth2.Config
	RajaOngkir  RajaOngkirConfig
}

func LoadConfig() AppConfig {
	if err := godotenv.Load(); err != nil {
		// ignore - env vars might already be set in production
		fmt.Fprintln(os.Stderr, "environment variable failed to load or .env file not found")
		os.Exit(1)
	}

	originCityID, _ := strconv.Atoi(os.Getenv("RAJAONGKIR_ORIGIN_CITY_ID"))

	cfg := AppConfig{
		BaseURL:   os.Getenv("BASE_URL"),
		Port:      os.Getenv("PORT"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		AppEnv:    os.Getenv("APP_ENV"),
		SMTP: email.SMTPConfig{
			AppName:     os.Getenv("APP_NAME"),
			Host:        os.Getenv("SMTP_HOST"),
			Port:        os.Getenv("SMTP_PORT"),
			AuthUser:    os.Getenv("SMTP_USERNAME"),
			AuthPass:    os.Getenv("SMTP_PASSWORD"),
			LogoURL:     os.Getenv("EMAIL_LOGO_URL"),
			FromName:    os.Getenv("EMAIL_FROM_NAME"),
			FromEmail:   os.Getenv("EMAIL_FROM_ADDRESS"),
			UseTLS:      getEnvBool("SMTP_USE_TLS", false),
			FrontendURL: os.Getenv("FRONTEND_URL"),
		},
		GoogleOAuth: &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
		RajaOngkir: RajaOngkirConfig{
			APIKey:   os.Getenv("RAJAONGKIR_API_KEY"),
			BaseURL:  os.Getenv("RAJAONGKIR_BASE_URL"),
			OriginID: originCityID,
		},
	}

	if cfg.RajaOngkir.BaseURL == "" {
		cfg.RajaOngkir.BaseURL = "https://api.rajaongkir.com/starter"
	}
	cfg.RajaOngkir.BaseURL = strings.TrimSuffix(cfg.RajaOngkir.BaseURL, "/")

	// validate required fields
	if cfg.SMTP.AppName == "" || cfg.JWTSecret == "" || cfg.BaseURL == "" {
		fmt.Fprintln(os.Stderr, "fatal: required environment variables are missing: APP_NAME, JWT_SECRET, BASE_URL")
		os.Exit(1)
	}

	// JWTSecret must be at least 32 bytes for HS256
	if len(cfg.JWTSecret) < 32 {
		fmt.Fprintln(os.Stderr, "fatal: JWT_SECRET must be at least 32 characters long")
		os.Exit(1)
	}

	// validate SMTP config
	if cfg.SMTP.Host == "" || cfg.SMTP.Port == "" || cfg.SMTP.AuthUser == "" || cfg.SMTP.AuthPass == "" || cfg.SMTP.FromEmail == "" {
		fmt.Fprintln(os.Stderr, "fatal: SMTP configuration is incomplete: SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD, EMAIL_FROM_ADDRESS are required")
		os.Exit(1)
	}

	// validate Google OAuth config
	if cfg.GoogleOAuth.ClientID == "" || cfg.GoogleOAuth.ClientSecret == "" || cfg.GoogleOAuth.RedirectURL == "" {
		fmt.Fprintln(os.Stderr, "fatal: Google OAuth configuration is incomplete: GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GOOGLE_REDIRECT_URL are required")
		os.Exit(1)
	}

	// validate RajaOngkir config
	if cfg.RajaOngkir.APIKey == "" || cfg.RajaOngkir.OriginID <= 0 {
		fmt.Fprintln(os.Stderr, "fatal: RajaOngkir configuration is incomplete: RAJAONGKIR_API_KEY and valid RAJAONGKIR_ORIGIN_CITY_ID are required")
		os.Exit(1)
	}

	// normalize FrontendURL (remove trailing slash)
	cfg.SMTP.FrontendURL = strings.TrimSuffix(cfg.SMTP.FrontendURL, "/")

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg
}

func getEnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return strings.ToLower(val) == "true" || val == "1"
}
