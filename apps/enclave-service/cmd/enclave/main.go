package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
)

type HealthResponse struct {
	Status string `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime string `json:"uptime"`
}

var startTime time.Time

func main() {
	startTime = time.Now()
	log.Println("Starting enclave service...")

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
		listener.Close()
		os.Exit(0)
	}()
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		log.Println("New connection accepted")
		go handleConnection(conn)
	}
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
