package main

import (
	"log"
	"os"

	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/delivery/http/middleware"
	"github.com/bubblevy/bookstore/internal/infrastructure/pgsql"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/bubblevy/bookstore/pkg/jwt"
	"github.com/bubblevy/bookstore/pkg/validator"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Environment variable failed to load or .env file not found")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	appName := os.Getenv("APP_NAME")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USERNAME")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	baseURL := os.Getenv("BASE_URL")
	emailFromName := os.Getenv("EMAIL_FROM_NAME")
	emailFromAddress := os.Getenv("EMAIL_FROM_ADDRESS")
	frontendURL := os.Getenv("FRONTEND_URL")
	if appName == "" || jwtSecret == "" || smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPass == "" || baseURL == "" || emailFromName == "" || emailFromAddress == "" || frontendURL == "" {
		log.Fatal("One or more required environment variables are missing")
	}

	smtpCfg := email.SMTPConfig{
		AppName:     appName,
		Host:        smtpHost,
		Port:        smtpPort,
		AuthUser:    smtpUser,
		AuthPass:    smtpPass,
		LogoURL:     "https://e3bd-160-22-24-14.ngrok-free.app/api/v1/static/images/logo.png",
		FromName:    emailFromName,
		FromEmail:   emailFromAddress,
		FrontendURL: frontendURL,
	}

	pgdb, err := pgsql.Init()
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	app := fiber.New()
	api := app.Group("/api/v1")
	api.Get("/static/*", static.New("./public"))

	jwtService := jwt.NewJWTService(jwtSecret)
	validate := validator.NewValidator()
	authMiddleware := middleware.AuthMiddleware(jwtService)
	emailSender := email.NewSMTPEmailSender(smtpCfg)

	userRepo := pgsql.NewUserRepository(pgdb)
	sessionRepo := pgsql.NewSessionRepository(pgdb)
	authUsecase := usecase.NewAuthUsecase(userRepo, sessionRepo, jwtService, emailSender)
	handler.NewAuthHandler(api, authUsecase, validate, authMiddleware)

	log.Println("Server berjalan di " + baseURL)
	log.Fatal(app.Listen(baseURL[7:]))
}
