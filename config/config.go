package config

import (
	"log"
	"os"

	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/joho/godotenv"
)

type AppConfig struct {
	BaseURL   string
	Port      string
	JWTSecret string
	SMTP      email.SMTPConfig
}

func LoadConfig() AppConfig {
	if err := godotenv.Load(); err != nil {
		log.Println("environment variable failed to load or .env file not found")
	}

	cfg := AppConfig{
		BaseURL:   os.Getenv("BASE_URL"),
		Port:      os.Getenv("PORT"),
		JWTSecret: os.Getenv("JWT_SECRET"),
		SMTP: email.SMTPConfig{
			AppName:     os.Getenv("APP_NAME"),
			Host:        os.Getenv("SMTP_HOST"),
			Port:        os.Getenv("SMTP_PORT"),
			AuthUser:    os.Getenv("SMTP_USERNAME"),
			AuthPass:    os.Getenv("SMTP_PASSWORD"),
			LogoURL:     "https://e3bd-160-22-24-14.ngrok-free.app/api/v1/static/images/logo.png",
			FromName:    os.Getenv("EMAIL_FROM_NAME"),
			FromEmail:   os.Getenv("EMAIL_FROM_ADDRESS"),
			FrontendURL: os.Getenv("FRONTEND_URL"),
		},
	}

	if cfg.SMTP.AppName == "" || cfg.JWTSecret == "" || cfg.BaseURL == "" {
		log.Fatal("one or more required environment variables are missing")
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg
}
