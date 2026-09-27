package main

import (
	"log"
	"time"

	"github.com/bubblevy/bookstore/config"
	httpDelivery "github.com/bubblevy/bookstore/internal/delivery/http"
	"github.com/bubblevy/bookstore/internal/infrastructure/pgsql"

	"github.com/gofiber/fiber/v3"
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

	// setup route & dependency
	httpDelivery.SetupRoutes(app, pgdb, cfg)

	// start server
	log.Printf("server berjalan di port :%s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
