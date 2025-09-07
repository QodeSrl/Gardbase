package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/QodeSrl/gardbase-api/internal/handlers"
	"github.com/QodeSrl/gardbase-api/internal/middleware"
	"github.com/QodeSrl/gardbase-api/internal/storage"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	router *gin.Engine
	logger *zap.Logger
	config *Config	
}

type Config struct {
	Port string `env:"PORT" envDefault:"8080"`
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	config := loadConfig()

	server := NewServer(config, logger)

	server.setupRoutes()

	server.start()
}

func NewServer(config *Config, logger *zap.Logger) *Server {
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(middleware.GinZapLogger(logger))
	router.Use(middleware.GinZapRecovery(logger, true))
	router.Use(middleware.CorsMiddleware())
	router.Use(middleware.RateLimitMiddleware(logger))

	return &Server{
		router: router,
		logger: logger,
		config: config,
	}
}

func (s *Server) setupRoutes() {
	s.router.GET("/health", handlers.HealthCheckHandler)

	api := s.router.Group("/api")

	ctx := context.Background()
	s3Client, dynamoClient, err := initStorage(ctx)
	if err != nil {
		s.logger.Fatal("Failed to initialize storage clients", zap.Error(err))
	}
	
	objectHandler := handlers.NewObjectHandler(s3Client, dynamoClient)
	objects := api.Group("/objects")
	objects.Use(middleware.TenantMiddleware())
	{
		objects.GET("/:id", objectHandler.Get)
		objects.POST("", objectHandler.Create)
	}
}

func (s *Server) start() {
	srv := &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: s.router,
	}

	// start server in a goroutine
	go func() {
		s.logger.Info("Starting Gardbase API server", 
			zap.String("port", s.config.Port),
			zap.String("environment", s.config.Environment))
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	s.logger.Info("Server exited")
}

func initStorage(ctx context.Context) (*storage.S3Client, *storage.DynamoClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	bucket := getEnvOrPanic("S3_BUCKET")
	objectsTable := getEnvOrPanic("DYNAMO_OBJECTS_TABLE")
	indexesTable := getEnvOrPanic("DYNAMO_INDEXES_TABLE")

	s3Client := storage.NewS3Client(ctx, bucket, cfg)
	dynamoClient := storage.NewDynamoClient(ctx, objectsTable, indexesTable, cfg)

	return s3Client, dynamoClient, nil
} 

func loadConfig() *Config {
	config := &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	return config
}

func getEnv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
func getEnvOrPanic(key string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	panic("environment variable " + key + " is required")
}