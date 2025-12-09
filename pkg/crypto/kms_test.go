package crypto

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"os"
	"testing"
	"time"
)

var config SessionConfig
var keyID string

// Note: These tests assume that there is a running enclave proxy at the specified endpoint.
// Other untested functions (e.g., SessionUnwrap, UnsealDEK) would require more complex setup and are to be tested separately in SDKs, possibly with integration tests.

func TestMain(m *testing.M) {
	var enclaveEndpoint string
	flag.StringVar(&enclaveEndpoint, "enclave-endpoint", "", "Enclave proxy endpoint URL")
	config = SessionConfig{
		Endpoint:          enclaveEndpoint,
		MaxAttestationAge: 5 * time.Minute,
		VerifyPCRs:        false,
		HTTPTimeout:       10 * time.Second,
	}
	flag.StringVar(&keyID, "key-id", "", "Key ID for KMS")
	flag.Parse()
	os.Exit(m.Run())
}

func TestStartDecryptSession(t *testing.T) {
	ctx := context.Background()
	sess, err := StartDecryptSession(ctx, config)
	if err != nil {
		t.Fatalf("Failed to start decrypt session: %v", err)
	}
	t.Logf("Successfully started decrypt session: %+v", sess)
}

func TestGenerateDEK(t *testing.T) {
	ctx := context.Background()
	sess, err := StartDecryptSession(ctx, config)
	if err != nil {
		t.Fatalf("Failed to start decrypt session: %v", err)
	}
	DEKs, encryptedDEKS, err := sess.GenerateDEK(ctx, keyID, 3)
	if err != nil {
		t.Fatalf("Failed to generate DEK: %v", err)
	}
	if len(DEKs) != 3 || len(encryptedDEKS) != 3 {
		t.Fatalf("Expected 3 DEKs and encrypted DEKs, got %d and %d", len(DEKs), len(encryptedDEKS))
	}
	t.Logf("Successfully generated DEKs and encrypted DEKs")
}

func TestUnwrapSingleDEK(t *testing.T) {
	ctx := context.Background()
	sess, err := StartDecryptSession(ctx, config)
	if err != nil {
		t.Fatalf("Failed to start decrypt session: %v", err)
	}
	DEKs, encryptedDEKS, err := sess.GenerateDEK(ctx, keyID, 1)
	if err != nil {
		t.Fatalf("Failed to generate DEK: %v", err)
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}
	unwrappedDEK, err := UnwrapSingleDEK(ctx, config.Endpoint, base64.StdEncoding.EncodeToString(encryptedDEKS[0]), base64.StdEncoding.EncodeToString(nonce), keyID)
	if err != nil {
		t.Fatalf("Failed to unwrap DEK: %v", err)
	}
	if !bytes.Equal(DEKs[0], unwrappedDEK) {
		t.Fatalf("Unwrapped DEK does not match original DEK")
	}
	t.Logf("Successfully unwrapped DEK")
}
