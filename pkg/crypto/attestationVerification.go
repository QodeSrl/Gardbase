package crypto

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"time"
)

type attestationDocument struct {
	ModuleID    string                 `cbor:"module_id"`
	Timestamp   uint64                 `cbor:"timestamp"`
	Digest      string                 `cbor:"digest"`
	PCRs        map[uint][]byte        `cbor:"pcrs"`
	Certificate []byte                 `cbor:"certificate"`
	CABundle    [][]byte               `cbor:"cabundle"`
	PublicKey   []byte                 `cbor:"public_key,omitempty"`
	UserData    []byte                 `cbor:"user_data,omitempty"`
	Nonce       []byte                 `cbor:"nonce,omitempty"`
	Protected   map[string]interface{} `cbor:"protected"`
	Signature   []byte                 `cbor:"signature"`
}

// See: https://docs.aws.amazon.com/enclaves/latest/user/verify-root.html#COSE-CBOR
type coseSign1 struct {
	_           struct{} `cbor:",toarray"`
	Protected   []byte
	Unprotected map[interface{}]interface{}
	Payload     []byte
	Signature   []byte
}

type verificationResult struct {
	Document      *attestationDocument
	PCRs          map[uint][]byte
	PublicKey     []byte
	Nonce         []byte
	Timestamp     time.Time
	Verified      bool
	VerifiedSteps []string
}

// AWS Nitro Enclaves Root CA certificate
// this is the public root certificate used to verify the attestation certificate chain
// see: https://docs.aws.amazon.com/enclaves/latest/user/verify-root.html#validation-process
const awsNitroRootCA = `-----BEGIN CERTIFICATE-----
MIICETCCAZagAwIBAgIRAPkxdWgbkK/hHUbMtOTn+FYwCgYIKoZIzj0EAwMwSTEL
MAkGA1UEBhMCVVMxDzANBgNVBAoMBkFtYXpvbjEMMAoGA1UECwwDQVdTMRswGQYD
VQQDDBJhd3Mubml0cm8tZW5jbGF2ZXMwHhcNMTkxMDI4MTMyODA1WhcNNDkxMDI4
MTQyODA1WjBJMQswCQYDVQQGEwJVUzEPMA0GA1UECgwGQW1hem9uMQwwCgYDVQQL
DANBV1MxGzAZBgNVBAMMEmF3cy5uaXRyby1lbmNsYXZlczB2MBAGByqGSM49AgEG
BSuBBAAiA2IABPwCVOumCMHzaHDimtqQvkY4MpJzbolL//Zy2YlES1BR5TSksfbb
48C8WBoyt7F2Bw7eEtaaP+ohG2bnUs990d0JX28TcPQXCEPZ3BABIeTPYwEoCWZE
h8l5YoQwTcU/9KNCMEAwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUkCW1DdkF
R+eWw5b6cp3PmanfS5YwDgYDVR0PAQH/BAQDAgGGMAoGCCqGSM49BAMDA2kAMGYC
MQCjfy+Rocm9Xue4YnwWmNJVA44fA0P5W2OpYow9OYCVRaEevL8uO1XYru5xtMPW
rfMCMQCi85sWBbJwKKXdS6BptQFuZbT73o/gBh1qUxl/nNr12UO8Yfwr6wPLb+6N
IwLz3/Y=
-----END CERTIFICATE-----`

func (ds *DecryptSession) verifyAttestation(config SessionConfig) (*verificationResult, error) {
	result := &verificationResult{
		VerifiedSteps: make([]string, 0),
	}

	attDoc, err := base64.StdEncoding.DecodeString(ds.AttestationB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode attestation document: %w", err)
	}

	// 1: decode COSE Sign1 structure

	// 2: decode attestation document from payload

	// 3: verify certificate chain

	// 4: verify signature

	// 5: verify timestamp

	// 6: verify nonce

	// 7: verify public key

	// 8: verify PCRs

	return result, nil
}

// Verifies the certificate chain up to the root CA
func verifyCertificateChain(leafCertDER []byte, caBundleDER [][]byte, rootCA *x509.Certificate) error {
	return nil
}

func verifyCOSESignature(cose *coseSign1, leafCertDER []byte) error {
	return nil
}

func verifyPCRs(actualPCRs map[uint][]byte, expectedPCRs map[uint]string) error {
	return nil
}
