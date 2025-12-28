package main

import (
	"bufio"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/handlers"
	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/hf/nsm"
	"github.com/hf/nsm/request"
	"github.com/mdlayher/vsock"
)

var (
	startTime time.Time
	kmsClient *kms.Client
)

var (
	nsmSession        *nsm.Session
	nsmPrivateKey     *rsa.PrivateKey
	nsmPublicKeyBytes []byte
	nsmAttestation    []byte
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

	port := getEnvUint32("ENCLAVE_PORT", 5000)

	listener, err := vsock.Listen(port, nil)
	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Enclave service is listening on port %d", port)

	// grateful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down enclave service...")
		if nsmSession != nil {
			nsmSession.Close()
			nsmSession = nil
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

func initiateNSM() (err error) {
	// open NSM session
	nsmSession, err = nsm.OpenDefaultSession()
	if err != nil {
		return fmt.Errorf("failed to open NSM session: %v", err)
	}

	// generate RSA key pair for NSM session, nsmSession is used here as a crypto/rand.Reader
	key, err := rsa.GenerateKey(nsmSession, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %v", err)
	}
	nsmPrivateKey = key

	// extract public key in DER format
	nsmPublicKeyBytes, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal NSM public key: %v", err)
	}

	// request attestation document with nsm public key
	req := request.Attestation{
		PublicKey: nsmPublicKeyBytes,
		Nonce:     nil,
		UserData:  nil,
	}
	res, err := nsmSession.Send(&req)
	if err != nil {
		return fmt.Errorf("failed to get attestation document: %v", err)
	}
	if res.Attestation == nil || len(res.Attestation.Document) == 0 {
		return fmt.Errorf("received empty attestation document from NSM")
	}
	nsmAttestation = res.Attestation.Document

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

		var req enclaveproto.Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			log.Printf("Failed to unmarshal request: %v", err)
			utils.SendError(encoder, fmt.Sprintf("Invalid JSON: %v", err))
			continue
		}

		log.Printf("Received request: %+v", req)

		switch req.Type {
		case "health":
			handlers.HandleHealth(encoder, startTime)
		case "get_attestation":
			handlers.HandleGetAttestation(encoder, nsmAttestation)
		case "session_init":
			handlers.HandleSessionInit(encoder, req.Payload, nsmSession)
		case "session_unwrap":
			handlers.HandleSessionUnwrap(encoder, req.Payload, nsmSession, nsmPrivateKey, nsmAttestation, kmsClient)
		case "session_prepare_dek":
			handlers.HandleSessionPrepareDEK(encoder, req.Payload, nsmSession, nsmPrivateKey)
		case "decrypt":
			handlers.HandleDecrypt(encoder, req.Payload, nsmSession, kmsClient, nsmPublicKeyBytes, nsmPrivateKey)
		default:
			log.Printf("Unknown request type: %s", req.Type)
			utils.SendError(encoder, fmt.Sprintf("Unknown request type: %s", req.Type))
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from connection: %v", err)
		utils.SendError(encoder, fmt.Sprintf("Error reading from connection: %v", err))
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvUint32(key string, defaultValue uint32) uint32 {
	if valueStr := os.Getenv(key); valueStr != "" {
		if uintVal, err := strconv.ParseUint(valueStr, 10, 32); err == nil {
			return uint32(uintVal)
		}
	}
	return defaultValue
}
