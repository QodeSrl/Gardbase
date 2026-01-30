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
	flag.Parse()
	os.Exit(m.Run())
}

func TestInitEnclaveSecureSession(t *testing.T) {
	ctx := context.Background()
	sess, err := InitEnclaveSecureSession(ctx, config)
	if err != nil {
		t.Fatalf("Failed to start decrypt session: %v", err)
	}
	t.Logf("Successfully started decrypt session: %+v", sess)
}

func TestGenerateDEK(t *testing.T) {
	ctx := context.Background()
	sess, err := InitEnclaveSecureSession(ctx, config)
	if err != nil {
		t.Fatalf("Failed to start decrypt session: %v", err)
	}
	DEKs, err := sess.GenerateDEK(ctx, 3)
	if err != nil {
		t.Fatalf("Failed to generate DEK: %v", err)
	}
	if len(DEKs) != 3 {
		t.Fatalf("Expected 3 DEKs, got %d", len(DEKs))
	}
	t.Logf("Successfully generated DEKs")
}

func TestUnwrapSingleDEK(t *testing.T) {
	ctx := context.Background()
	sess, err := InitEnclaveSecureSession(ctx, config)
	if err != nil {
		t.Fatalf("Failed to start decrypt session: %v", err)
	}
	DEKs, err := sess.GenerateDEK(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to generate DEK: %v", err)
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("Failed to generate nonce: %v", err)
	}
	unwrappedDEK, err := UnwrapSingleDEK(ctx, config.Endpoint, base64.StdEncoding.EncodeToString(DEKs[0].KMSEncryptedDEK), base64.StdEncoding.EncodeToString(nonce))
	if err != nil {
		t.Fatalf("Failed to unwrap DEK: %v", err)
	}
	if !bytes.Equal(DEKs[0].PlaintextDEK, unwrappedDEK) {
		t.Fatalf("Unwrapped DEK does not match original DEK")
	}
	t.Logf("Successfully unwrapped DEK")
}
