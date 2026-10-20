package http

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bubblevy/bookstore/config"
	"github.com/bubblevy/bookstore/internal/delivery/http/handler"
	"github.com/bubblevy/bookstore/internal/delivery/http/middleware"
	"github.com/bubblevy/bookstore/internal/infrastructure/pgsql"
	"github.com/bubblevy/bookstore/internal/usecase"
	"github.com/bubblevy/bookstore/pkg/biteship"
	"github.com/bubblevy/bookstore/pkg/cache"
	"github.com/bubblevy/bookstore/pkg/email"
	"github.com/bubblevy/bookstore/pkg/jwt"
	"github.com/bubblevy/bookstore/pkg/logger"
	"github.com/bubblevy/bookstore/pkg/payment"
	"github.com/bubblevy/bookstore/pkg/rabbitmq"
	"github.com/bubblevy/bookstore/pkg/storage"
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
	app.Use(middleware.HTTPLogger())
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(middleware.CORS([]string{cfg.SMTP.FrontendURL}))

	// root
	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Hello World!",
		})
	})

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
		healthStatus := fiber.Map{
			"status":    "ok",
			"database":  "connected",
			"timestamp": time.Now().UTC(),
		}
		if cfg.Redis.Enabled {
			healthStatus["redis"] = "enabled"
		}
		if cfg.RabbitMQ.Enabled {
			healthStatus["rabbitmq"] = "enabled"
		}
		return c.JSON(healthStatus)
	})

	// init dependencies & services
	jwtService := jwt.NewJWTService(cfg.JWTSecret)
	validate := validator.NewValidator()
	authMiddleware := middleware.AuthMiddleware(jwtService)
	adminMiddleware := middleware.AdminMiddleware()
	emailSender := email.NewSMTPEmailSender(cfg.SMTP)
	txManager := pgsql.NewTransactionManager(db)
	midtransSvc := payment.NewMidtransService(cfg.Midtrans.ServerKey, cfg.Midtrans.IsProduction)

	// init storage service
	storageSvc, err := storage.NewStorageService(cfg.Storage)
	if err != nil {
		logger.Error("failed initializing storage service", "error", err)
		os.Exit(1)
	}

	// init redis cache
	cacheSvc, err := cache.NewRedisCache(cfg.Redis)
	if err != nil {
		logger.Warn("redis cache initialization notice", "error", err)
	}

	// init rabbitmq email queue worker
	var activeEmailSender email.EmailSender = emailSender
	rabbitClient, err := rabbitmq.NewRabbitMQClient(cfg.RabbitMQ)
	if err != nil {
		logger.Warn("rabbitmq initialization notice, falling back to direct smtp", "error", err)
	} else if rabbitClient != nil {
		activeEmailSender = rabbitmq.NewRabbitMQEmailSender(rabbitClient, emailSender)
		if err := rabbitClient.StartEmailConsumer(context.Background(), emailSender); err != nil {
			logger.Warn("failed to start rabbitmq email consumer", "error", err)
		}
	}

	userRepo := pgsql.NewUserRepository(db)
	sessionRepo := pgsql.NewSessionRepository(db)
	authUsecase := usecase.NewAuthUsecase(userRepo, sessionRepo, txManager, jwtService, activeEmailSender)

	// start background session cleanup (every 6 hours)
	authUsecase.StartSessionCleanup(context.Background(), 6*time.Hour)

	// routing group
	api := app.Group("/api/v1")
	api.Get("/static/*", static.New(cfg.Storage.LocalBaseDir))

	// Serve static asset folders directly from root domain for STORAGE_LOCAL_URL_PREFIX=/ or full URL configurations
	app.Get("/covers/*", static.New(filepath.Join(cfg.Storage.LocalBaseDir, "covers")))
	app.Get("/categories/*", static.New(filepath.Join(cfg.Storage.LocalBaseDir, "categories")))
	app.Get("/uploads/*", static.New(filepath.Join(cfg.Storage.LocalBaseDir, "uploads")))
	app.Get("/temp/*", static.New(filepath.Join(cfg.Storage.LocalBaseDir, "temp")))

	cleanPrefix := strings.TrimSpace(cfg.Storage.LocalURLPrefix)
	if cleanPrefix != "" && cleanPrefix != "/" && cleanPrefix != "/api/v1/static" && !strings.HasPrefix(cleanPrefix, "http://") && !strings.HasPrefix(cleanPrefix, "https://") {
		trimmed := strings.TrimRight(cleanPrefix, "/")
		app.Get(trimmed+"/*", static.New(cfg.Storage.LocalBaseDir))
	}

	// uploads routes (Admin)
	uploadUsecase := usecase.NewUploadUsecase(storageSvc, db)
	// start background orphaned file cleanup (runs every 12 hours, removes abandoned files older than 2 hours)
	uploadUsecase.StartOrphanedFilesCleanup(context.Background(), 12*time.Hour, 2*time.Hour)
	handler.NewUploadHandler(api, uploadUsecase, authMiddleware, adminMiddleware)

	// auth routes
	authGroup := api.Group("/auth")
	handler.NewAuthHandler(authGroup, authUsecase, validate, authMiddleware, cfg)

	// user routes (/me and Admin User Management)
	userUsecase := usecase.NewUserUsecase(userRepo)
	handler.NewUserHandler(api, userUsecase, authMiddleware, adminMiddleware)

	// address routes
	addressRepo := pgsql.NewAddressRepository(db)
	addressUsecase := usecase.NewAddressUsecase(db, addressRepo)
	handler.NewAddressHandler(api, addressUsecase, validate, authMiddleware)

	// biteship clients & webhook auth
	webhookBiteshipAuth := middleware.BiteshipWebhookAuth(cfg.Biteship.WebhookSecret)
	biteshipClient := biteship.NewClient(cfg.Biteship.APIKey, cfg.Biteship.BaseURL, cfg.Biteship.OriginAreaID)

	// category routes
	categoryRepo := pgsql.NewCategoryRepository(db)
	categoryUsecase := usecase.NewCategoryUsecaseWithCache(categoryRepo, storageSvc, cacheSvc)
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
	bookUsecase := usecase.NewBookUsecaseWithCache(bookRepo, categoryRepo, authorRepo, publisherRepo, storageSvc, cacheSvc)
	handler.NewBookHandler(api, bookUsecase, validate, authMiddleware, adminMiddleware)

	// discount routes
	discountRepo := pgsql.NewDiscountRepository(db)
	discountUsecase := usecase.NewDiscountUsecase(discountRepo, bookRepo)
	handler.NewDiscountHandler(api, discountUsecase, validate, authMiddleware, adminMiddleware)

	// cart routes
	cartRepo := pgsql.NewCartRepository(db)
	cartUsecase := usecase.NewCartUsecase(cartRepo, bookRepo)
	handler.NewCartHandler(api, cartUsecase, validate, authMiddleware)

	// checkout & order routes
	checkoutRepo := pgsql.NewCheckoutRepository(db)
	orderRepo := pgsql.NewOrderRepository(db)

	checkoutUsecase := usecase.NewCheckoutUsecase(db, checkoutRepo, orderRepo, addressRepo, discountUsecase, discountRepo, biteshipClient, midtransSvc)
	handler.NewCheckoutHandler(api, checkoutUsecase, validate, authMiddleware)

	orderUsecase := usecase.NewOrderUsecase(orderRepo)
	handler.NewOrderHandler(api, orderUsecase, authMiddleware)

	// shipping routes
	shipmentRepo := pgsql.NewShipmentRepository(db)
	shipmentUseCase := usecase.NewShippingUsecase(db, orderRepo, shipmentRepo, checkoutRepo, addressRepo, biteshipClient, cfg.Biteship.ToStoreShippingConfig())
	handler.NewShippingHandler(api, shipmentUseCase, validate, authMiddleware, adminMiddleware, webhookBiteshipAuth)
}
