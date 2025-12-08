package crypto

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"
)

var config SessionConfig
var keyID string

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
