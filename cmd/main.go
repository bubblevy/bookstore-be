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
	"github.com/gofiber/fiber/v3/middleware/compress"
)

func main() {
	cfg := config.LoadConfig()

	// Init logger
	var logPath string
	if cfg.AppEnv == "production" || cfg.AppEnv == "prod" {
		logPath = os.Getenv("LOG_PATH")
		if logPath == "" {
			logPath = "logs/app.log"
		}
	}
	logger.Init(cfg.AppEnv, logPath)

	// CRUCIAL: Jalankan background scheduler HANYA di Master Process
	// Mencegah scheduler berjalan 4x di setiap child worker
	if !fiber.IsChild() {
		logger.StartRetentionScheduler(context.Background(), logger.GetLogDir(), 14*24*time.Hour, 24*time.Hour)
	}

	// Timezone
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		logger.Error("failed to load timezone Asia/Jakarta", "error", err)
		os.Exit(1)
	}
	time.Local = loc

	// Init database
	pgdb, err := pgsql.Init()
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Init Fiber instance
	app := fiber.New(fiber.Config{
		AppName:         cfg.SMTP.AppName,
		ServerHeader:    "",
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		Concurrency:     256 * 1024,
		CaseSensitive:   true,
		StrictRouting:   true,
	})

	// Kompresi otomatis
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Setup routes
	httpDelivery.SetupRoutes(app, pgdb, cfg)

	// Graceful shutdown handler
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", "port", cfg.Port, "app", cfg.SMTP.AppName, "env", cfg.AppEnv)
		listenCfg := fiber.ListenConfig{
			EnablePrefork: true,
		}

		if err := app.Listen(":"+cfg.Port, listenCfg); err != nil {
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
