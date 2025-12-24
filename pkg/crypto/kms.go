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
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/nacl/box"
)

type EnclaveSecureSession struct {
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

func InitEnclaveSecureSession(ctx context.Context, config SessionConfig) (*EnclaveSecureSession, error) {

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

	var resBody enclaveproto.Response[enclaveproto.SessionInitResponse]
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, err
	}
	data := resBody.Data

	enclavePubRaw, err := base64.StdEncoding.DecodeString(data.EnclaveEphemeralPublicKey)
	if err != nil {
		return nil, errors.New("invalid enclave ephemeral public key")
	}
	if len(enclavePubRaw) != 32 {
		return nil, fmt.Errorf("invalid enclave ephemeral public key length")
	}

	expiresAt, err := time.Parse(time.RFC3339, data.ExpiresAt)
	if err != nil {
		return nil, errors.New("invalid session expiration time")
	}

	sessionKey, err := deriveSessionKey(clientPriv, enclavePubRaw)
	if err != nil {
		return nil, err
	}

	ess := &EnclaveSecureSession{
		SessionId:           data.SessionId,
		ClientPriv:          clientPriv,
		ClientPub:           clientPub,
		EnclavePubRaw:       enclavePubRaw,
		SessionKey:          sessionKey,
		ExpiresAt:           expiresAt,
		AttestationB64:      data.Attestation,
		ExpectedNonceB64:    nonceB64,
		httpClient:          httpClient,
		endpoint:            config.Endpoint,
		AttestationVerified: false,
	}

	if _, err := ess.verifyAttestation(config); err != nil {
		zero(ess.SessionKey)
		return nil, fmt.Errorf("attestation verification failed: %w", err)
	}

	return ess, nil
}

func (ess *EnclaveSecureSession) SessionUnwrap(ctx context.Context, items []enclaveproto.SessionUnwrapItem, keyID string) (enclaveproto.SessionUnwrapResponse, error) {
	if time.Now().After(ess.ExpiresAt) {
		return nil, errors.New("decrypt session has expired")
	}
	if !ess.AttestationVerified {
		return nil, errors.New("attestation not verified")
	}

	body := enclaveproto.SessionUnwrapRequest{
		SessionId: ess.SessionId,
		KeyId:     keyID,
		Items:     items,
	}
	reqBytes, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ess.endpoint+"/session/unwrap", strings.NewReader(string(reqBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := ess.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to unwrap session items: status %d", res.StatusCode)
	}

	var resBody enclaveproto.Response[enclaveproto.SessionUnwrapResponse]
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, err
	}
	return resBody.Data, nil
}

type GeneratedDEK struct {
	PlaintextDEK []byte
	EncryptedDEK []byte
}

func (ess *EnclaveSecureSession) GenerateDEK(ctx context.Context, keyID string, count int) (generatedDEKs []GeneratedDEK, err error) {
	if time.Now().After(ess.ExpiresAt) {
		return nil, errors.New("decrypt session has expired")
	}
	if !ess.AttestationVerified {
		return nil, errors.New("attestation not verified")
	}

	body := enclaveproto.SessionGenerateDEKRequest{
		SessionId: ess.SessionId,
		KeyId:     keyID,
		Count:     count,
	}
	reqBytes, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ess.endpoint+"/session/generate-deks", strings.NewReader(string(reqBytes)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := ess.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to unwrap session items: status %d", res.StatusCode)
	}

	var resBody enclaveproto.Response[enclaveproto.SessionGenerateDEKResponse]
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, err
	}
	for _, DEK := range resBody.Data.DEKs {
		dek, err := openDEK(ess.SessionKey, DEK.SealedDEK, DEK.Nonce)
		if err != nil {
			return nil, err
		}
		encryptedDEK, err := base64.StdEncoding.DecodeString(DEK.KmsEncryptedDEK)
		if err != nil {
			return nil, errors.New("invalid base64 KMS encrypted DEK")
		}
		generatedDEKs = append(generatedDEKs, GeneratedDEK{
			PlaintextDEK: dek,
			EncryptedDEK: encryptedDEK,
		})
	}
	return generatedDEKs, nil
}

func (ess *EnclaveSecureSession) UnsealDEK(ctx context.Context, encryptedDEKB64 string, nonceB64 string, objectID string) ([]byte, error) {
	if ess.SessionKey == nil || len(ess.SessionKey) != chacha20poly1305.KeySize {
		return nil, errors.New("invalid session key")
	}

	dek, err := openDEK(ess.SessionKey, encryptedDEKB64, nonceB64)
	if err != nil {
		return nil, err
	}

	return dek, nil
}

// Close zeros out sensitive data
func (ess *EnclaveSecureSession) Close() {
	zero(ess.SessionKey)
	zero(ess.ClientPriv[:])
}

func (ess *EnclaveSecureSession) GetAttestationInfo() map[string]any {
	if ess.AttestationResult == nil {
		return nil
	}

	pcrInfo := make(map[string]string)
	for idx, value := range ess.AttestationResult.PCRs {
		pcrInfo[fmt.Sprintf("PCR%d", idx)] = hex.EncodeToString(value)
	}

	return map[string]interface{}{
		"verified":       ess.AttestationVerified,
		"timestamp":      ess.AttestationResult.Timestamp,
		"pcrs":           pcrInfo,
		"verified_steps": ess.AttestationResult.VerifiedSteps,
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
	var resBody enclaveproto.Response[enclaveproto.DecryptResponse]
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, err
	}
	data := resBody.Data
	ciphertextBox, err := base64.StdEncoding.DecodeString(data.Ciphertext)
	if err != nil {
		return nil, errors.New("invalid base64 ciphertext")
	}
	enclavePubRaw, err := base64.StdEncoding.DecodeString(data.EnclavePubKey)
	if err != nil {
		return nil, errors.New("invalid base64 enclave public key")
	}
	if len(enclavePubRaw) != 32 {
		return nil, errors.New("invalid enclave public key length")
	}
	var nonce [24]byte
	copy(nonce[:], ciphertextBox[:24])

	// TODO: verify attestation in data.Attestation

	dek, ok := box.Open(nil, ciphertextBox[24:], &nonce, (*[32]byte)(enclavePubRaw), clientPriv)
	if !ok {
		return nil, errors.New("failed to decrypt DEK with NaCl box")
	}
	return dek, nil
}
