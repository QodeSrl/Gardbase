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
	"github.com/QodeSrl/gardbase/apps/api/internal/services"
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
	Port        string
	Environment string
}

type AWSConfig struct {
	Region                   string
	S3Bucket                 string
	DynamoObjectsTable       string
	DynamoIndexesTable       string
	DynamoTenantConfigsTable string
	DynamoAPIKeysTable       string
	KMSKeyID                 string
	MaxRetries               int
	RequestTimeout           time.Duration
	UseLocalstack            bool
	LocalstackUrl            string
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	config := loadConfig()

	server := NewServer(config, logger)

	s3Client, dynamoClient, kmsService, err := initAWSServices(context.Background(), logger)
	if err != nil {
		logger.Fatal("Failed to initialize storage clients", zap.Error(err))
	}

	server.setupRoutes(s3Client, dynamoClient, kmsService)
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

func (s *Server) setupRoutes(s3Client *storage.S3Client, dynamoClient *storage.DynamoClient, kmsService *services.KMS) {
	vsock := &services.Vsock{
		EnclaveCID:  getEnvUint32("ENCLAVE_CID", 16),
		EnclavePort: getEnvUint32("ENCLAVE_PORT", 8080),
	}

	api := s.router.Group("/api")

	healthCheckHandler := &handlers.HealthCheckHandler{
		Vsock:    vsock,
		KMS:      kmsService,
		Dynamo:   dynamoClient,
		S3Client: s3Client,
	}
	health := api.Group("/health")
	health.GET("/", healthCheckHandler.HandleAPIHealthCheck)
	health.GET("/enclave", healthCheckHandler.HandleEnclaveHealthCheck)
	health.GET("/storage", healthCheckHandler.HandleStorageHealthCheck)
	health.GET("/kms", healthCheckHandler.HandleKMSHealthCheck)

	tenantHandler := &handlers.TenantHandler{
		Vsock:  vsock,
		Dynamo: dynamoClient,
		KMS:    kmsService,
	}
	tenants := api.Group("/tenants")
	tenants.POST("/", tenantHandler.HandleCreateTenant)

	objectHandler := &handlers.ObjectHandler{
		BaseURL:    getEnv("BASE_URL", "https://api.gardbase.com") + "/api",
		Vsock:      vsock,
		S3Client:   s3Client,
		Dynamo:     dynamoClient,
		KMS:        kmsService,
		PresignTTL: 15 * time.Minute,
	}
	objects := api.Group("/objects")
	objects.Use(middleware.TenantMiddleware(dynamoClient))
	objects.POST("/table-hash", objectHandler.GetTableHash)
	objects.GET("/:table-hash/:id", objectHandler.Get)
	objects.POST("/:table-hash", objectHandler.Create)
	objects.PUT("/:table-hash/:id/upload-inline", objectHandler.UploadInline)

	encryptionHandler := &handlers.EncryptionHandler{
		Vsock:  vsock,
		Dynamo: dynamoClient,
		KMS:    kmsService,
	}
	encryption := api.Group("/encryption")
	encryption.Use(middleware.TenantMiddleware(dynamoClient))
	encryption.POST("/secure-session/init", encryptionHandler.HandleSessionInit)
	encryption.POST("/secure-session/unwrap", encryptionHandler.HandleSessionUnwrap)
	encryption.POST("/secure-session/generate-deks", encryptionHandler.HandleSessionGenerateDEK)
	encryption.POST("/decrypt", encryptionHandler.HandleDecrypt)
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

func initAWSServices(ctx context.Context, logger *zap.Logger) (*storage.S3Client, *storage.DynamoClient, *services.KMS, error) {
	awsConfig := loadAWSConfig()

	cfg, err := loadAWSSDKConfig(ctx, awsConfig)
	if err != nil {
		return nil, nil, nil, err
	}

	s3Client := storage.NewS3Client(ctx, awsConfig.S3Bucket, cfg, awsConfig.UseLocalstack, awsConfig.LocalstackUrl)
	dynamoClient := storage.NewDynamoClient(ctx, awsConfig.DynamoObjectsTable, awsConfig.DynamoIndexesTable, awsConfig.DynamoTenantConfigsTable, awsConfig.DynamoAPIKeysTable, cfg, awsConfig.UseLocalstack, awsConfig.LocalstackUrl)
	kmsService := services.NewKMSService(ctx, cfg, awsConfig.KMSKeyID, awsConfig.UseLocalstack, awsConfig.LocalstackUrl)

	if err := testAWSConnectivity(ctx, s3Client, dynamoClient, logger); err != nil {
		return nil, nil, nil, err
	}

	return s3Client, dynamoClient, kmsService, nil
}

func loadConfig() *Config {
	config := &Config{
		Port:        getEnv("PORT", "80"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	return config
}

func loadAWSConfig() *AWSConfig {
	return &AWSConfig{
		Region:                   getEnv("AWS_REGION", "eu-central-1"),
		S3Bucket:                 getEnvOrPanic("S3_BUCKET"),
		DynamoObjectsTable:       getEnvOrPanic("DYNAMO_OBJECTS_TABLE"),
		DynamoIndexesTable:       getEnvOrPanic("DYNAMO_INDEXES_TABLE"),
		DynamoTenantConfigsTable: getEnvOrPanic("DYNAMO_TENANT_CONFIGS_TABLE"),
		DynamoAPIKeysTable:       getEnvOrPanic("DYNAMO_API_KEYS_TABLE"),
		KMSKeyID:                 getEnvOrPanic("KMS_KEY_ID"),
		MaxRetries:               getEnvAsInt("AWS_MAX_RETRIES", 3),
		RequestTimeout:           time.Duration(getEnvAsInt("AWS_REQUEST_TIMEOUT", 5)) * time.Second,
		UseLocalstack:            getEnvAsBool("USE_LOCALSTACK", false),
		LocalstackUrl:            getEnv("LOCALSTACK_URL", "http://localhost:4566"),
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
