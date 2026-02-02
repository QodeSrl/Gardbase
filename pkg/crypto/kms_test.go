package crypto

import (
	"context"
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
