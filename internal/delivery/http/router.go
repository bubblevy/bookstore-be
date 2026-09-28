package http

import (
	"context"
	"time"

	"github.com/bubblevy/bookstore/config"
	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/delivery/http/middleware"
	"github.com/bubblevy/bookstore/internal/infrastructure/pgsql"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/bubblevy/bookstore/pkg/jwt"
	"github.com/bubblevy/bookstore/pkg/validator"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.App, db *gorm.DB, cfg config.AppConfig) {
	// global middleware
	app.Use(middleware.RequestID())
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(middleware.CORS([]string{cfg.SMTP.FrontendURL}))

	// health check
	app.Get("/health", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"service":   cfg.SMTP.AppName,
			"timestamp": time.Now().UTC(),
		})
	})

	app.Get("/health/ready", func(c fiber.Ctx) error {
		sqlDB, err := db.DB()
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status":    "error",
				"reason":    "database connection unavailable",
				"timestamp": time.Now().UTC(),
			})
		}
		if err := sqlDB.Ping(); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status":    "error",
				"reason":    "database ping failed",
				"timestamp": time.Now().UTC(),
			})
		}
		return c.JSON(fiber.Map{
			"status":    "ok",
			"database":  "connected",
			"timestamp": time.Now().UTC(),
		})
	})

	// init dependencies
	jwtService := jwt.NewJWTService(cfg.JWTSecret)
	validate := validator.NewValidator()
	authMiddleware := middleware.AuthMiddleware(jwtService)
	emailSender := email.NewSMTPEmailSender(cfg.SMTP)

	userRepo := pgsql.NewUserRepository(db)
	sessionRepo := pgsql.NewSessionRepository(db)
	authUsecase := usecase.NewAuthUsecase(userRepo, sessionRepo, jwtService, emailSender)

	// start background session cleanup (every 6 hours)
	authUsecase.StartSessionCleanup(context.Background(), 6*time.Hour)

	// routing
	api := app.Group("/api/v1")
	api.Get("/static/*", static.New("./public"))

	// auth routes
	authGroup := api.Group("/auth")
	authGroup.Use(middleware.RateLimit(middleware.AuthRateLimit))
	handler.NewAuthHandler(authGroup, authUsecase, validate, authMiddleware)

	// user routes
	userGroup := api.Group("/users")
	userGroup.Use(middleware.RateLimit(middleware.StandardRateLimit))
	// handler.NewUserHandler(userGroup, ...)

	// public routes - relaxed rate limit (200 req/min)
	// publicGroup := api.Group("/public")
	// publicGroup.Use(middleware.RateLimit(middleware.ReadRateLimit))
	// handler.NewPublicHandler(publicGroup, ...)
}
