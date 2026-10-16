package http

import (
	"context"
	"time"

	"github.com/bubblevy/bookstore/config"
	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/delivery/http/middleware"
	"github.com/bubblevy/bookstore/internal/infrastructure/pgsql"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/bubblevy/bookstore/pkg/jwt"
	"github.com/bubblevy/bookstore/pkg/payment"
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
	adminMiddleware := middleware.AdminMiddleware()
	emailSender := email.NewSMTPEmailSender(cfg.SMTP)
	txManager := pgsql.NewTransactionManager(db)
	midtransSvc := payment.NewMidtransService(cfg.Midtrans.ServerKey, cfg.Midtrans.IsProduction)

	userRepo := pgsql.NewUserRepository(db)
	sessionRepo := pgsql.NewSessionRepository(db)
	authUsecase := usecase.NewAuthUsecase(userRepo, sessionRepo, txManager, jwtService, emailSender)

	// start background session cleanup (every 6 hours)
	authUsecase.StartSessionCleanup(context.Background(), 6*time.Hour)

	// routing
	api := app.Group("/api/v1")
	api.Get("/static/*", static.New("./public"))

	// auth routes
	authGroup := api.Group("/auth")
	handler.NewAuthHandler(authGroup, authUsecase, validate, authMiddleware, cfg)

	// address routes
	addressRepo := pgsql.NewAddressRepository(db)
	addressUsecase := usecase.NewAddressUsecase(db, addressRepo)
	handler.NewAddressHandler(api, addressUsecase, validate, authMiddleware)

	// biteship
	biteshipClient := biteship.NewClient(cfg.Biteship.APIKey, cfg.Biteship.BaseURL, cfg.Biteship.OriginAreaID)

	// category routes
	categoryRepo := pgsql.NewCategoryRepository(db)
	categoryUsecase := usecase.NewCategoryUsecase(categoryRepo)
	handler.NewCategoryHandler(api, categoryUsecase, validate, authMiddleware, adminMiddleware)

	// publisher routes
	publisherRepo := pgsql.NewPublisherRepository(db)
	publisherUsecase := usecase.NewPublisherUsecase(publisherRepo)
	handler.NewPublisherHandler(api, publisherUsecase, validate, authMiddleware, adminMiddleware)

	// author routes
	authorRepo := pgsql.NewAuthorRepository(db)
	authorUsecase := usecase.NewAuthorUsecase(authorRepo)
	handler.NewAuthorHandler(api, authorUsecase, validate, authMiddleware, adminMiddleware)

	// book routes
	bookRepo := pgsql.NewBookRepository(db)
	bookUsecase := usecase.NewBookUsecase(bookRepo, categoryRepo, authorRepo, publisherRepo)
	handler.NewBookHandler(api, bookUsecase, validate, authMiddleware, adminMiddleware)

	// discount routes
	discountRepo := pgsql.NewDiscountRepository(db)
	discountUsecase := usecase.NewDiscountUsecase(discountRepo, bookRepo)
	handler.NewDiscountHandler(api, discountUsecase, validate, authMiddleware, adminMiddleware)

	// cart routes
	cartRepo := pgsql.NewCartRepository(db)
	cartUsecase := usecase.NewCartUsecase(cartRepo, bookRepo)
	handler.NewCartHandler(api, cartUsecase, validate, authMiddleware)

	// order routes
	orderRepo := pgsql.NewOrderRepository(db)
	checkoutUsecase := usecase.NewCheckoutUsecase(db, orderRepo, cartRepo, addressRepo, discountRepo, biteshipClient, midtransSvc)
	handler.NewCheckoutHandler(api, checkoutUsecase, validate, authMiddleware)

	// shipping routes
	shipmentRepo := pgsql.NewShipmentRepository(db)
	shipmentUseCase := usecase.NewShippingUsecase(db, orderRepo, shipmentRepo, biteshipClient, cfg.Biteship.ToStoreShippingConfig())
	handler.NewShippingHandler(api, shipmentUseCase, authMiddleware, adminMiddleware)
}
