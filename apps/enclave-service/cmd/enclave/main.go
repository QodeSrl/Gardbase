package main

import (
	"bufio"
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/handlers"
	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/hf/nsm"
	"github.com/mdlayher/vsock"
)

var (
	startTime time.Time
)

var (
	nsmSession        *nsm.Session
	nsmPrivateKey     *rsa.PrivateKey
	nsmPublicKeyBytes []byte

	attestation utils.Attestation = utils.Attestation{Mu: sync.RWMutex{}}
)

func main() {
	startTime = time.Now()
	log.Println("Starting enclave service...")

	if err := initiateNSM(); err != nil {
		log.Fatalf("Failed to initiate NSM session: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	startAttestationRefresher(ctx)

	port := getEnvUint32("ENCLAVE_PORT", 5000)

	listener, err := vsock.Listen(port, nil)
	if err != nil {
		log.Fatalf("Failed to start listener: %v", err)
	}
	defer listener.Close()

	log.Printf("Enclave service is listening on port %d", port)

	// graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		cancel()
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

func initiateNSM() (err error) {
	// open NSM session
	nsmSession, err = nsm.OpenDefaultSession()
	if err != nil {
		return fmt.Errorf("failed to open NSM session: %v", err)
	}

	// generate RSA key pair for NSM session, nsmSession is used here as a crypto/rand.Reader
	nsmPrivateKey, err = rsa.GenerateKey(nsmSession, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %v", err)
	}

	// extract public key in DER format
	nsmPublicKeyBytes, err = x509.MarshalPKIXPublicKey(&nsmPrivateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal RSA public key: %v", err)
	}

	log.Printf(
		"RSA public key fingerprint: %x",
		sha256.Sum256(nsmPublicKeyBytes),
	)

	log.Println("NSM session initiated")

	return refreshAttestation()
}

func refreshAttestation() error {
	att, err := utils.RequestAttestation(nsmSession, nsmPublicKeyBytes)
	if err != nil {
		return err
	}

	attestation.Mu.Lock()
	attestation.Doc = append([]byte(nil), att...)
	attestation.Mu.Unlock()

	log.Printf("Attestation document refreshed: len=%d", len(attestation.Doc))
	return nil
}

func startAttestationRefresher(ctx context.Context) {
	ticker := time.NewTicker(4 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := refreshAttestation(); err != nil {
					log.Printf("Failed to refresh attestation document: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
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

		log.Printf("Received request type: %s, payload size: %d bytes", req.Type, len(req.Payload))

		switch req.Type {
		case "health":
			handlers.HandleHealth(encoder, startTime)
		case "get_attestation":
			handlers.HandleGetAttestation(encoder, &attestation)
		case "session_init":
			handlers.HandleSessionInit(encoder, req.Payload, nsmSession)
		case "session_unwrap":
			handlers.HandleSessionUnwrap(encoder, req.Payload, nsmSession, nsmPrivateKey)
		case "session_prepare_dek":
			handlers.HandleSessionPrepareDEK(encoder, req.Payload, nsmSession, nsmPrivateKey)
		case "decrypt":
			handlers.HandleDecrypt(encoder, req.Payload, nsmPrivateKey)
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
