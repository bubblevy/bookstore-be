package main

import (
	"log"
	"os"

	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/delivery/http/middleware"
	"github.com/bubblevy/bookstore/internal/infrastructure/pgsql"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/jwt"
	"github.com/bubblevy/bookstore/pkg/validator"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Environment variable failed to load or .env file not found")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is missing")
	}

	pgdb, err := pgsql.Init()
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	app := fiber.New()
	api := app.Group("/api/v1")

	jwtService := jwt.NewJWTService(jwtSecret)
	validate := validator.NewValidator()
	authMiddleware := middleware.AuthMiddleware(jwtService)

	userRepo := pgsql.NewUserRepository(pgdb)
	sessionRepo := pgsql.NewSessionRepository(pgdb)
	authUsecase := usecase.NewAuthUsecase(userRepo, sessionRepo, jwtService)
	handler.NewAuthHandler(api, authUsecase, validate, authMiddleware)

	log.Println("Server berjalan di http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}
