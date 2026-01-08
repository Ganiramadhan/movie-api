package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "movie-backend/docs"
	"movie-backend/internal/config"
	"movie-backend/internal/database"
	"movie-backend/internal/handlers"
	"movie-backend/internal/repository"
	"movie-backend/internal/routes"
	"movie-backend/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// @title Movie Backend API
// @version 1.0
// @description Backend API untuk konsumsi TMDB API, manajemen data film, dashboard analytics, dan chart visualization
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://github.com/yourusername/movie-backend
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8010
// @BasePath /api/v1
// @schemes http https

func main() {
	// Load environment variables
	loadEnvFile()

	// Load configuration
	cfg := config.Load()

	// Setup logger
	log := setupLogger()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Warnf("Configuration validation warning: %v", err)
	}

	// Connect to database
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Errorf("Error closing database connection: %v", err)
		}
	}()

	movieRepo := repository.NewMovieRepository(db)
	genreRepo := repository.NewGenreRepository(db)
	langRepo := repository.NewLanguageRepository(db)
	movieService := services.NewMovieService(movieRepo, genreRepo, langRepo, cfg, log)
	movieHandler := handlers.NewMovieHandler(movieService, log)

	minioService, err := services.NewMinIOService(&cfg.MinIO, log)
	if err != nil {
		log.Fatalf("Failed to initialize MinIO service: %v", err)
	}

	if ms, ok := movieService.(interface{ SetMinIOService(*services.MinIOService) }); ok {
		ms.SetMinIOService(minioService)
	}

	uploadHandler := handlers.NewUploadHandler(minioService, log)

	app := fiber.New(fiber.Config{
		AppName:               "Movie Backend API",
		ReadTimeout:           cfg.Server.ReadTimeout,
		WriteTimeout:          cfg.Server.WriteTimeout,
		IdleTimeout:           120 * time.Second,
		DisableStartupMessage: false,
		ErrorHandler:          customErrorHandler(log),
	})

	setupMiddleware(app)

	app.Get("/health", healthCheckHandler(db))

	// Swagger documentation
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Setup API routes
	routes.Setup(app, movieHandler, uploadHandler)

	// Graceful shutdown
	go gracefulShutdown(app, log)

	log.Infof("Movie Backend API starting on port %s", cfg.Server.Port)
	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

func setupLogger() *logrus.Logger {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.InfoLevel)

	if os.Getenv("GO_ENV") == "dev" || os.Getenv("GO_ENV") == "development" {
		log.SetLevel(logrus.DebugLevel)
	}

	return log
}

func setupMiddleware(app *fiber.App) {
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	// Logger middleware
	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path} | ${error}\n",
		TimeFormat: "15:04:05",
		TimeZone:   "Local",
	}))

	// CORS middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS, PATCH",
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}))
}

func healthCheckHandler(db *database.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		dbStatus := "healthy"
		if err := db.HealthCheck(); err != nil {
			dbStatus = "unhealthy"
		}

		return c.JSON(fiber.Map{
			"status":    "ok",
			"service":   "movie-backend",
			"version":   "1.0.0",
			"database":  dbStatus,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func customErrorHandler(log *logrus.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		log.WithError(err).WithFields(logrus.Fields{
			"method": c.Method(),
			"path":   c.Path(),
			"status": code,
		}).Error("Request error")

		return c.Status(code).JSON(fiber.Map{
			"status":  "error",
			"code":    code,
			"message": err.Error(),
		})
	}
}

func gracefulShutdown(app *fiber.App, log *logrus.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		log.Errorf("Error during shutdown: %v", err)
	}

	log.Info("Server shutdown complete")
}

func loadEnvFile() {
	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{})
	log.SetOutput(os.Stdout)

	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "dev"
	}

	execDir, err := os.Getwd()
	if err != nil {
		log.Warnf("Could not get working directory: %v", err)
		return
	}

	envFile := filepath.Join(execDir, "envs", ".env."+env)
	if err := godotenv.Load(envFile); err != nil {
		log.Warnf("Could not load environment file %s: %v", envFile, err)

		defaultEnvFile := filepath.Join(execDir, "envs", ".env")
		if err := godotenv.Load(defaultEnvFile); err != nil {
			log.Warnf("Could not load default environment file: %v", err)
		} else {
			log.Infof("Environment loaded from default file %s", defaultEnvFile)
		}
	} else {
		log.Infof("Environment loaded from file %s", envFile)
	}
}
