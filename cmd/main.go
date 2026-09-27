package main

import (
	"log"
	"time"

	"github.com/bubblevy/bookstore/config"
	httpDelivery "github.com/bubblevy/bookstore/internal/delivery/http"
	"github.com/bubblevy/bookstore/internal/infrastructure/pgsql"
	"github.com/bubblevy/bookstore/pkg/response"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

func main() {
	cfg := config.LoadConfig()

	// timezone
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Fatalf("failed to load timezone Asia/Jakarta: %v", err)
	}
	time.Local = loc

	// init database
	pgdb, err := pgsql.Init()
	if err != nil {
		log.Fatalf("failed to connect to the database: %v", err)
	}

	// init fiber
	app := fiber.New(fiber.Config{
		AppName: cfg.SMTP.AppName,
	})

	app.Use(recover.New())
	app.Use(helmet.New())

	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.SMTP.FrontendURL},
		AllowCredentials: true,
	}))

	app.Use(limiter.New(limiter.Config{
		Max:        3,
		Expiration: 1 * time.Minute,
		LimitReached: func(c fiber.Ctx) error {
			return response.Error(c, fiber.StatusTooManyRequests, "Too many requests", nil)
		},
	}))

	// setup route & dependency
	httpDelivery.SetupRoutes(app, pgdb, cfg)

	// start server
	log.Printf("server berjalan di port :%s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
