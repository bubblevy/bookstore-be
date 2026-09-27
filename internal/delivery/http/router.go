package http

import (
	"github.com/bubblevy/bookstore/config"
	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/delivery/http/middleware"
	"github.com/bubblevy/bookstore/internal/infrastructure/pgsql"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/bubblevy/bookstore/pkg/jwt"
	"github.com/bubblevy/bookstore/pkg/validator"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.App, db *gorm.DB, cfg config.AppConfig) {
	// global middleware (security & recovery)
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.SMTP.FrontendURL},
		AllowCredentials: true,
	}))

	// init dependencies
	jwtService := jwt.NewJWTService(cfg.JWTSecret)
	validate := validator.NewValidator()
	authMiddleware := middleware.AuthMiddleware(jwtService)
	emailSender := email.NewSMTPEmailSender(cfg.SMTP)

	userRepo := pgsql.NewUserRepository(db)
	sessionRepo := pgsql.NewSessionRepository(db)
	authUsecase := usecase.NewAuthUsecase(userRepo, sessionRepo, jwtService, emailSender)

	// routing
	api := app.Group("/api/v1")
	api.Get("/static/*", static.New("./public"))

	// init handler
	handler.NewAuthHandler(api, authUsecase, validate, authMiddleware)
	// handler.NewBookHandler(...)
	// handler.NewOrderHandler(...)
}
