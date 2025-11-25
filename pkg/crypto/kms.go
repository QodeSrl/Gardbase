package crypto

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/nacl/box"
)

type DecryptSession struct {
	SessionId           string   // Base64-encoded
	ClientPriv          [32]byte // x25519 private key
	ClientPub           [32]byte // x25519 public key
	EnclavePubRaw       []byte   // enclave's ephemeral public key (decoded)
	SessionKey          []byte   // derived session key
	ExpiresAt           time.Time
	AttestationB64      string // Base64-encoded attestation document
	ExpectedNonceB64    string // Base64-encoded expected nonce for attestation
	AttestationVerified bool
	AttestationResult   *verificationResult
	endpoint            string
	httpClient          *http.Client
}

type SessionConfig struct {
	// base URL of the proxy server
	Endpoint string
	// PCR values you expect from the enclave
	// Key: PCR index; Value: hex-encoded PCR hash
	// Note: use "nitro-cli describe-eif --eif-path enclave.eif" to get these
	ExpectedPCRs map[uint]string
	// AWS Nitro Root CA certificate (optional)
	RootCA *x509.Certificate
	// Maximum age of the attestation document
	MaxAttestationAge time.Duration
	// Whether to verify PCR values
	// Set to false during development, true in production
	VerifyPCRs bool
	// HTTPTimeout for requests
	HTTPTimeout time.Duration
}

func GenerateDEK(ctx context.Context, kmsClient *kms.Client, keyID string) (DEK []byte, encryptedDEK []byte, err error) {
	generateDEKInput := &kms.GenerateDataKeyInput{
		KeyId:   &keyID,
		KeySpec: "AES_256",
	}
	generateDEKOutput, err := kmsClient.GenerateDataKey(ctx, generateDEKInput)
	if err != nil {
		return nil, nil, err
	}
	return generateDEKOutput.Plaintext, generateDEKOutput.CiphertextBlob, nil
}

func StartDecryptSession(ctx context.Context, config SessionConfig) (*DecryptSession, error) {

	clientPriv, clientPub, clientPubB64, err := GenerateEphemeralKeypair()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	nonceB64 := base64.StdEncoding.EncodeToString(nonce)

	reqBody := enclaveproto.SessionInitRequest{
		ClientEphemeralPublicKey: clientPubB64,
		Nonce:                    nonceB64,
	}
	reqBytes, _ := json.Marshal(reqBody)
	httpClient := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.Endpoint+"/session/init", strings.NewReader(string(reqBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to start decrypt session: status %d", res.StatusCode)
	}

	var resBody enclaveproto.SessionInitResponse
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, err
	}

	enclavePubRaw, err := base64.StdEncoding.DecodeString(resBody.EnclaveEphemeralPublicKey)
	if err != nil {
		return nil, errors.New("invalid enclave ephemeral public key")
	}
	if len(enclavePubRaw) != 32 {
		return nil, errors.New("invalid enclave ephemeral public key length")
	}

	expiresAt, err := time.Parse(time.RFC3339, resBody.ExpiresAt)
	if err != nil {
		return nil, errors.New("invalid session expiration time")
	}

	sessionKey, err := deriveSessionKey(clientPriv, enclavePubRaw)
	if err != nil {
		return nil, err
	}

	ds := &DecryptSession{
		SessionId:           resBody.SessionId,
		ClientPriv:          clientPriv,
		ClientPub:           clientPub,
		EnclavePubRaw:       enclavePubRaw,
		SessionKey:          sessionKey,
		ExpiresAt:           expiresAt,
		AttestationB64:      resBody.Attestation,
		ExpectedNonceB64:    nonceB64,
		httpClient:          httpClient,
		AttestationVerified: false,
	}

	if _, err := ds.verifyAttestation(config); err != nil {
		zero(ds.SessionKey)
		return nil, fmt.Errorf("attestation verification failed: %w", err)
	}

	return ds, nil
}

func (ds *DecryptSession) SessionUnwrap(ctx context.Context, items []enclaveproto.SessionUnwrapItem, keyID string) (enclaveproto.SessionUnwrapResponse, error) {
	if time.Now().After(ds.ExpiresAt) {
		return nil, errors.New("decrypt session has expired")
	}
	if !ds.AttestationVerified {
		return nil, errors.New("attestation not verified")
	}

	body := enclaveproto.SessionUnwrapRequest{
		SessionId: ds.SessionId,
		KeyId:     keyID,
		Items:     items,
	}
	reqBytes, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ds.endpoint, strings.NewReader(string(reqBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := ds.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to unwrap session items: status %d", res.StatusCode)
	}

	var resBody enclaveproto.SessionUnwrapResponse
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, err
	}
	return resBody, nil
}

func (ds *DecryptSession) UnsealDEK(ctx context.Context, encryptedDEKB64 string, nonceB64 string, objectID string) ([]byte, error) {
	if ds.SessionKey == nil || len(ds.SessionKey) != chacha20poly1305.KeySize {
		return nil, errors.New("invalid session key")
	}

	encryptedDEKBytes, err := base64.StdEncoding.DecodeString(encryptedDEKB64)
	if err != nil {
		return nil, errors.New("invalid base64 encrypted DEK")
	}
	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return nil, errors.New("invalid base64 nonce")
	}
	if len(nonce) != chacha20poly1305.NonceSizeX {
		return nil, errors.New("invalid nonce size")
	}

	aead, err := chacha20poly1305.NewX(ds.SessionKey)
	if err != nil {
		return nil, err
	}
	dek, err := aead.Open(nil, nonce, encryptedDEKBytes, []byte(objectID))
	if err != nil {
		return nil, err
	}
	return dek, nil
}

// Close zeros out sensitive data
func (ds *DecryptSession) Close() {
	zero(ds.SessionKey)
	zero(ds.ClientPriv[:])
}

func (ds *DecryptSession) GetAttestationInfo() map[string]any {
	if ds.AttestationResult == nil {
		return nil
	}

	pcrInfo := make(map[string]string)
	for idx, value := range ds.AttestationResult.PCRs {
		pcrInfo[fmt.Sprintf("PCR%d", idx)] = hex.EncodeToString(value)
	}

	return map[string]interface{}{
		"verified":       ds.AttestationVerified,
		"timestamp":      ds.AttestationResult.Timestamp,
		"pcrs":           pcrInfo,
		"verified_steps": ds.AttestationResult.VerifiedSteps,
	}
}

func UnwrapSingleDEK(ctx context.Context, endpoint string, wrappedDEKB64 string, nonceB64 string, keyID string) ([]byte, error) {
	clientPub, clientPriv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	clientPubB64 := base64.StdEncoding.EncodeToString(clientPub[:])

	body := enclaveproto.DecryptRequest{
		Ciphertext:               wrappedDEKB64,
		Nonce:                    nonceB64,
		KeyID:                    keyID,
		ClientEphemeralPublicKey: clientPubB64,
	}

	reqBytes, _ := json.Marshal(body)
	httpClient := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(reqBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to unwrap DEK: status %d", res.StatusCode)
	}
	var resBody enclaveproto.DecryptResponse
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, err
	}
	ciphertextBox, err := base64.StdEncoding.DecodeString(resBody.Ciphertext)
	if err != nil {
		return nil, errors.New("invalid base64 ciphertext")
	}
	enclavePubRaw, err := base64.StdEncoding.DecodeString(resBody.EnclavePubKey)
	if err != nil {
		return nil, errors.New("invalid base64 enclave public key")
	}
	if len(enclavePubRaw) != 32 {
		return nil, errors.New("invalid enclave public key length")
	}
	var nonce [24]byte
	copy(nonce[:], ciphertextBox[:24])

	// TODO: verify attestation in resBody.Attestation

	dek, ok := box.Open(nil, ciphertextBox[24:], &nonce, (*[32]byte)(enclavePubRaw), clientPriv)
	if !ok {
		return nil, errors.New("failed to decrypt DEK with NaCl box")
	}
	return dek, nil
}
