package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bubblevy/bookstore/config"
	httpDelivery "github.com/bubblevy/bookstore/internal/delivery/http"
	"github.com/bubblevy/bookstore/internal/infrastructure/pgsql"
	"github.com/bubblevy/bookstore/pkg/logger"

	"github.com/gofiber/fiber/v3"
)

func main() {
	cfg := config.LoadConfig()

	// init logger - file output only in production
	var logPath string
	if cfg.AppEnv == "production" || cfg.AppEnv == "prod" {
		logPath = os.Getenv("LOG_PATH")
		if logPath == "" {
			logPath = "logs/app.log"
		}
	}
	logger.Init(cfg.AppEnv, logPath)
	// start background log retention scheduler (retention: 14 days, check every 24 hours)
	logger.StartRetentionScheduler(context.Background(), logger.GetLogDir(), 14*24*time.Hour, 24*time.Hour)

	// timezone
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		logger.Error("failed to load timezone Asia/Jakarta", "error", err)
		os.Exit(1)
	}
	time.Local = loc

	// init database
	pgdb, err := pgsql.Init()
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// init fiber
	app := fiber.New(fiber.Config{
		AppName: cfg.SMTP.AppName,
	})

	// setup route & dependency
	httpDelivery.SetupRoutes(app, pgdb, cfg)

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", "port", cfg.Port, "app", cfg.SMTP.AppName, "env", cfg.AppEnv)
		if err := app.Listen(":" + cfg.Port); err != nil {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Sync()
	logger.Info("server exited gracefully")
}
