package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/QodeSrl/gardbase/apps/api/internal/handlers"
	"github.com/QodeSrl/gardbase/apps/api/internal/middleware"
	"github.com/QodeSrl/gardbase/apps/api/internal/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Server struct {
	router *gin.Engine
	logger *zap.Logger
	config *Config
}

type Config struct {
	Port        string `env:"PORT" envDefault:"8080"`
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
}

type AWSConfig struct {
	Region             string
	S3Bucket           string
	DynamoObjectsTable string
	DynamoIndexesTable string
	MaxRetries         int
	RequestTimeout     time.Duration
	UseLocalstack      bool
	LocalstackUrl      string
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

	server.setupEnclaveProxy()

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
	s3Client, dynamoClient, err := initStorage(ctx, s.logger)
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

func (s *Server) setupEnclaveProxy() {
	proxy := &handlers.VsockProxy{
		EnclaveCID:  getEnvUint32("ENCLAVE_CID", 16),
		EnclavePort: getEnvUint32("ENCLAVE_PORT", 8080),
	}
	s.router.GET("/enclave/health", proxy.HandleHealth)
	s.router.POST("/enclave/session/init", proxy.HandleSessionInit)
	s.router.POST("/enclave/session/unwrap", proxy.HandleSessionUnwrap)
	s.router.POST("/enclave/decrypt", proxy.HandleDecrypt)
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

func initStorage(ctx context.Context, logger *zap.Logger) (*storage.S3Client, *storage.DynamoClient, error) {
	awsConfig := loadAWSConfig()

	cfg, err := loadAWSSDKConfig(ctx, awsConfig)
	if err != nil {
		return nil, nil, err
	}

	s3Client := storage.NewS3Client(ctx, awsConfig.S3Bucket, cfg, awsConfig.UseLocalstack, awsConfig.LocalstackUrl)
	dynamoClient := storage.NewDynamoClient(ctx, awsConfig.DynamoObjectsTable, awsConfig.DynamoIndexesTable, cfg, awsConfig.UseLocalstack, awsConfig.LocalstackUrl)

	if err := testAWSConnectivity(ctx, s3Client, dynamoClient, logger); err != nil {
		return nil, nil, err
	}

	return s3Client, dynamoClient, nil
}

func loadConfig() *Config {
	config := &Config{
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	return config
}

func loadAWSConfig() *AWSConfig {
	return &AWSConfig{
		Region:             getEnv("AWS_REGION", "eu-central-1"),
		S3Bucket:           getEnvOrPanic("S3_BUCKET"),
		DynamoObjectsTable: getEnvOrPanic("DYNAMO_OBJECTS_TABLE"),
		DynamoIndexesTable: getEnvOrPanic("DYNAMO_INDEXES_TABLE"),
		MaxRetries:         getEnvAsInt("AWS_MAX_RETRIES", 3),
		RequestTimeout:     time.Duration(getEnvAsInt("AWS_REQUEST_TIMEOUT", 5)) * time.Second,
		UseLocalstack:      getEnvAsBool("USE_LOCALSTACK", false),
		LocalstackUrl:      getEnv("LOCALSTACK_URL", "http://localhost:4566"),
	}
}

func loadAWSSDKConfig(ctx context.Context, awsConfig *AWSConfig) (aws.Config, error) {
	var configOpts []func(*config.LoadOptions) error

	configOpts = append(configOpts, config.WithRegion(awsConfig.Region))
	configOpts = append(configOpts, config.WithRetryMaxAttempts(awsConfig.MaxRetries))

	if awsConfig.UseLocalstack {
		configOpts = append(configOpts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")))
	}
	// else, the SDK will use the default credential provider chain

	cfg, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return aws.Config{}, err
	}
	return cfg, nil
}

func testAWSConnectivity(ctx context.Context, s3Client *storage.S3Client, dynamoClient *storage.DynamoClient, logger *zap.Logger) error {
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s3Client.TestConnnectivity(testCtx); err != nil {
		logger.Error("S3 connectivity test failed", zap.Error(err))
	}
	if err := dynamoClient.TestConnnectivity(testCtx); err != nil {
		logger.Error("DynamoDB connectivity test failed", zap.Error(err))
	}

	return nil
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
func getEnvAsInt(name string, defaultVal int) int {
	if valueStr := os.Getenv(name); valueStr != "" {
		if intVal, err := strconv.Atoi(valueStr); err == nil {
			return intVal
		}
	}
	return defaultVal
}
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
func getEnvUint32(key string, defaultValue uint32) uint32 {
	if value := os.Getenv(key); value != "" {
		if uintVal, err := strconv.ParseUint(value, 10, 32); err == nil {
			return uint32(uintVal)
		}
	}
	return defaultValue
}
