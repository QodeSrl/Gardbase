package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/hf/nsm"
)

type HealthResponse struct {
	Status string `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime string `json:"uptime"`
}

var (
	startTime time.Time
	kmsClient *kms.Client
	nsmSession *nsm.Session
)

func main() {
	startTime = time.Now()
	log.Println("Starting enclave service...")

	if err := initializeAWS(); err != nil {
		log.Fatalf("Failed to initialize AWS SDK: %v", err)
	}

	if err := initiateNSM(); err != nil {
		log.Fatalf("Failed to initiate NSM session: %v", err)
	}

	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	log.Println("Enclave service is listening on port 8000")

	// grateful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down enclave service...")
		if nsmSession != nil {
			nsmSession.Close()
		}
		listener.Close()
		os.Exit(0)
	}()
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		log.Printf("New connection accepted from %s", conn.RemoteAddr())
		go handleConnection(conn)
	}
}

func initializeAWS() error {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(getEnv("AWS_REGION", "eu-central-1")))
	if err != nil {
		return fmt.Errorf("unable to load AWS SDK config: %v", err)
	}

	kmsClient = kms.NewFromConfig(cfg)
	log.Println("AWS KMS client initialized")
	return nil
}

func initiateNSM() error {
	session, err := nsm.OpenDefaultSession()
	if err != nil {
		return fmt.Errorf("failed to open NSM session: %v", err)
	}
	nsmSession = session
	log.Println("NSM session initiated")
	return nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// connnection timeout
	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

	scanner := bufio.NewScanner(conn)
	encoder := json.NewEncoder(conn)

	for scanner.Scan() {
		// for each request, reset deadline
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		var req utils.Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			log.Printf("Failed to unmarshal request: %v", err)
			utils.SendError(encoder, fmt.Sprintf("Invalid JSON: %v", err))
			continue
		}

		log.Printf("Received request: %+v", req)

		switch req.Type {
		case "health":
			handleHealth(encoder)
		default:
			log.Printf("Unknown request type: %s", req.Type)
			utils.SendError(encoder, fmt.Sprintf("Unknown request type: %s", req.Type))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from connection: %v", err)
	}
}

func handleHealth(encoder *json.Encoder) {
	uptime := time.Since(startTime).String()
	res := utils.Response{
		Success: true,
		Data: HealthResponse{
			Status: "healthy",
			Timestamp: time.Now(),
			Uptime: uptime,
		},
		Message: "Service is healthy",
	}
	utils.SendResponse(encoder, res)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}