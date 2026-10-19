package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/bubblevy/bookstore/pkg/storage"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type BiteshipConfig struct {
	APIKey            string
	BaseURL           string
	OriginAreaID      string
	OriginPostalCode  int
	WebhookSecret     string
	AvailableCouriers string
	ShipperName       string
	ShipperPhone      string
	ShipperEmail      string
	OriginName        string
	OriginPhone       string
	OriginAddr        string
}

func (b BiteshipConfig) ToStoreShippingConfig() usecase.StoreShippingConfig {
	return usecase.StoreShippingConfig{
		ShipperName:       b.ShipperName,
		ShipperPhone:      b.ShipperPhone,
		ShipperEmail:      b.ShipperEmail,
		OriginName:        b.OriginName,
		OriginPhone:       b.OriginPhone,
		OriginAddr:        b.OriginAddr,
		OriginAreaID:      b.OriginAreaID,
		OriginPostalCode:  b.OriginPostalCode,
		AvailableCouriers: b.AvailableCouriers,
	}
}

type MidtransConfig struct {
	ServerKey    string
	ClientKey    string
	IsProduction bool
}

type AppConfig struct {
	BaseURL     string
	Port        string
	JWTSecret   string
	SMTP        email.SMTPConfig
	AppEnv      string
	GoogleOAuth *oauth2.Config
	Biteship    BiteshipConfig
	Midtrans    MidtransConfig
	Storage     storage.Config
}

func LoadConfig() AppConfig {
	if err := godotenv.Load(); err != nil {
		// .env file is optional when environment variables are set directly via environment/Docker
	}

	couriers := os.Getenv("BITESHIP_AVAILABLE_COURIERS")
	if couriers == "" {
		couriers = "jne,sicepat,jnt,anteraja,tiki,pos"
	}

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
		Biteship: BiteshipConfig{
			APIKey:            os.Getenv("BITESHIP_API_KEY"),
			BaseURL:           getEnv("BITESHIP_BASE_URL", "https://api.biteship.com"),
			OriginAreaID:      os.Getenv("BITESHIP_ORIGIN_AREA_ID"),
			OriginPostalCode:  getEnvInt("BITESHIP_ORIGIN_POSTAL_CODE", 59464),
			WebhookSecret:     os.Getenv("BITESHIP_WEBHOOK_SECRET"),
			AvailableCouriers: couriers,
			ShipperName:       getEnv("STORE_SHIPPER_NAME", "Admin Bookstore"),
			ShipperPhone:      os.Getenv("STORE_SHIPPER_PHONE"),
			ShipperEmail:      os.Getenv("STORE_SHIPPER_EMAIL"),
			OriginName:        getEnv("STORE_ORIGIN_NAME", "Gudang Utama Bookstore"),
			OriginPhone:       os.Getenv("STORE_ORIGIN_PHONE"),
			OriginAddr:        os.Getenv("STORE_ORIGIN_ADDRESS"),
		},
		Midtrans: MidtransConfig{
			ServerKey:    os.Getenv("MIDTRANS_SERVER_KEY"),
			ClientKey:    os.Getenv("MIDTRANS_CLIENT_KEY"),
			IsProduction: getEnvBool("MIDTRANS_IS_PRODUCTION", false),
		},
		Storage: storage.Config{
			Driver:         getEnv("STORAGE_DRIVER", "local"),
			LocalBaseDir:   getEnv("STORAGE_LOCAL_BASE_DIR", "./public"),
			LocalURLPrefix: getEnv("STORAGE_LOCAL_URL_PREFIX", "/api/v1/static"),
			S3: storage.S3Config{
				Bucket:          os.Getenv("AWS_S3_BUCKET"),
				Region:          getEnv("AWS_S3_REGION", "ap-southeast-1"),
				AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
				SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
				Endpoint:        os.Getenv("AWS_S3_ENDPOINT"),
				PublicCDNURL:    os.Getenv("AWS_S3_CDN_URL"),
				UsePathStyle:    getEnvBool("AWS_S3_USE_PATH_STYLE", false),
			},
		},
	}

	cfg.Biteship.BaseURL = strings.TrimSuffix(cfg.Biteship.BaseURL, "/")
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	// Ensure local storage URL prefix is a full absolute URL using BASE_URL
	if !strings.HasPrefix(cfg.Storage.LocalURLPrefix, "http://") && !strings.HasPrefix(cfg.Storage.LocalURLPrefix, "https://") {
		if cfg.BaseURL != "" {
			trimmedPrefix := strings.Trim(cfg.Storage.LocalURLPrefix, "/")
			if trimmedPrefix == "" {
				cfg.Storage.LocalURLPrefix = cfg.BaseURL
			} else {
				cfg.Storage.LocalURLPrefix = cfg.BaseURL + "/" + trimmedPrefix
			}
		}
	}

	if cfg.SMTP.AppName == "" || cfg.JWTSecret == "" || cfg.BaseURL == "" {
		fmt.Fprintln(os.Stderr, "fatal: required environment variables are missing: APP_NAME, JWT_SECRET, BASE_URL")
		os.Exit(1)
	}

	if len(cfg.JWTSecret) < 32 {
		fmt.Fprintln(os.Stderr, "fatal: JWT_SECRET must be at least 32 characters long")
		os.Exit(1)
	}

	if cfg.Biteship.APIKey == "" || cfg.Biteship.OriginAreaID == "" || cfg.Biteship.OriginPostalCode <= 0 {
		fmt.Fprintln(os.Stderr, "fatal: BITESHIP_API_KEY, BITESHIP_ORIGIN_AREA_ID, and STORE_ORIGIN_POSTAL_CODE are required")
		os.Exit(1)
	}

	if cfg.Biteship.OriginAddr == "" || cfg.Biteship.ShipperPhone == "" {
		fmt.Fprintln(os.Stderr, "fatal: STORE_ORIGIN_ADDRESS and STORE_SHIPPER_PHONE are required for shipping fulfillment")
		os.Exit(1)
	}

	cfg.SMTP.FrontendURL = strings.TrimSuffix(cfg.SMTP.FrontendURL, "/")
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return strings.ToLower(val) == "true" || val == "1"
}

func getEnvInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	v, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return fallback
	}
	return v
}
