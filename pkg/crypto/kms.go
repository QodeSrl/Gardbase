package crypto

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QodeSrl/gardbase/pkg/api/encryption"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/nacl/box"
)

type EnclaveSecureSession struct {
	SessionId           string
	ClientPriv          [32]byte // x25519 private key
	ClientPub           [32]byte // x25519 public key
	EnclavePubRaw       []byte   // enclave's ephemeral public key (decoded)
	SessionKey          []byte   // derived session key
	ExpiresAt           time.Time
	Attestation         []byte
	ExpectedNonce       []byte
	AttestationVerified bool
	AttestationResult   *verificationResult
	endpoint            string
	httpClient          *http.Client
}

type SessionConfig struct {
	// base URL of the proxy server
	Endpoint string
	// Tenant ID for authentication
	TenantID string
	// API Key for authentication
	APIKey string
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

type errBody struct {
	Error string `json:"error"`
}

type tenantRoundTripper struct {
	Base     http.RoundTripper
	TenantID string
	APIKey   string
}

func (t tenantRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("X-Tenant-ID", t.TenantID)
	req.Header.Set("X-API-Key", t.APIKey)
	return t.base().RoundTrip(req)
}

func (t tenantRoundTripper) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

func InitEnclaveSecureSession(ctx context.Context, config SessionConfig) (*EnclaveSecureSession, error) {
	clientPriv, clientPub, err := GenerateEphemeralKeypair()
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	reqBody := encryption.SessionInitRequest{
		ClientEphemeralPublicKey: clientPub[:],
		Nonce:                    nonce,
	}
	reqBytes, _ := json.Marshal(reqBody)
	httpClient := &http.Client{Timeout: 15 * time.Second, Transport: tenantRoundTripper{TenantID: config.TenantID, APIKey: config.APIKey}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.Endpoint+"/secure-session/init", strings.NewReader(string(reqBytes)))
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
		var errBody errBody
		if err := json.NewDecoder(res.Body).Decode(&errBody); err == nil {
			bodyBytes, _ := io.ReadAll(res.Body)
			return nil, fmt.Errorf("failed to start decrypt session: status %d, error: %s", res.StatusCode, string(bodyBytes))
		}
		return nil, fmt.Errorf("failed to start decrypt session: status %d", res.StatusCode)
	}

	var resBody encryption.SessionInitResponse
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, err
	}

	if len(resBody.EnclaveEphemeralPublicKey) != 32 {
		return nil, fmt.Errorf("invalid enclave ephemeral public key length")
	}

	expiresAt, err := time.Parse(time.RFC3339, resBody.ExpiresAt)
	if err != nil {
		return nil, errors.New("invalid session expiration time")
	}

	sessionKey, err := deriveSessionKey(clientPriv, resBody.EnclaveEphemeralPublicKey)
	if err != nil {
		return nil, err
	}

	ess := &EnclaveSecureSession{
		SessionId:           resBody.SessionId,
		ClientPriv:          clientPriv,
		ClientPub:           clientPub,
		EnclavePubRaw:       resBody.EnclaveEphemeralPublicKey,
		SessionKey:          sessionKey,
		ExpiresAt:           expiresAt,
		Attestation:         resBody.Attestation,
		ExpectedNonce:       nonce,
		httpClient:          httpClient,
		endpoint:            config.Endpoint,
		AttestationVerified: false,
	}

	if _, err := ess.verifyAttestation(config); err != nil {
		zero(ess.SessionKey)
		return ess, fmt.Errorf("attestation verification failed: %w", err)
	}

	ess.AttestationVerified = true

	return ess, nil
}

func (ess *EnclaveSecureSession) SessionUnwrap(ctx context.Context, items []enclaveproto.SessionUnwrapItem) (enclaveproto.SessionUnwrapResponse, error) {
	if time.Now().After(ess.ExpiresAt) {
		return nil, errors.New("decrypt session has expired")
	}
	if !ess.AttestationVerified {
		return nil, errors.New("attestation not verified")
	}

	body := encryption.SessionUnwrapRequest{
		SessionId: ess.SessionId,
		Items:     items,
	}
	reqBytes, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ess.endpoint+"/secure-session/unwrap", strings.NewReader(string(reqBytes)))
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
		var errBody errBody
		if err := json.NewDecoder(res.Body).Decode(&errBody); err == nil {
			bodyBytes, _ := io.ReadAll(res.Body)
			return nil, fmt.Errorf("failed to start decrypt session: status %d, error: %s", res.StatusCode, string(bodyBytes))
		}
		return nil, fmt.Errorf("failed to start decrypt session: status %d", res.StatusCode)
	}

	var resBody encryption.SessionUnwrapResponse
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, err
	}
	return resBody, nil
}

type GeneratedDEK struct {
	PlaintextDEK          []byte
	KMSEncryptedDEK       []byte
	MasterKeyEncryptedDEK []byte
	MasterKeyNonce        []byte
}

func (ess *EnclaveSecureSession) GenerateDEK(ctx context.Context, tableHash string, count int) (generatedDEKs []GeneratedDEK, iek []byte, err error) {
	if time.Now().After(ess.ExpiresAt) {
		return nil, nil, errors.New("decrypt session has expired")
	}
	if !ess.AttestationVerified {
		return nil, nil, errors.New("attestation not verified")
	}

	body := encryption.SessionGenerateDEKRequest{
		SessionId: ess.SessionId,
		Count:     count,
		TableHash: tableHash,
	}
	reqBytes, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ess.endpoint+"/secure-session/generate-deks", strings.NewReader(string(reqBytes)))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := ess.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		var errBody errBody
		if err := json.NewDecoder(res.Body).Decode(&errBody); err == nil {
			bodyBytes, _ := io.ReadAll(res.Body)
			return nil, nil, fmt.Errorf("failed to start decrypt session: status %d, error: %s", res.StatusCode, string(bodyBytes))
		}
		return nil, nil, fmt.Errorf("failed to start decrypt session: status %d", res.StatusCode)
	}

	var resBody encryption.SessionGenerateDEKResponse
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, nil, err
	}
	for _, DEK := range resBody.DEKs {
		dek, err := openDEK(ess.SessionKey, DEK.SealedDEK, DEK.SessionNonce, nil)
		if err != nil {
			return nil, nil, err
		}
		generatedDEKs = append(generatedDEKs, GeneratedDEK{
			PlaintextDEK:          dek,
			KMSEncryptedDEK:       DEK.KmsEncryptedDEK,
			MasterKeyEncryptedDEK: DEK.MasterEncryptedDEK,
			MasterKeyNonce:        DEK.MasterKeyNonce,
		})
	}
	iek, err = openDEK(ess.SessionKey, resBody.SealedIEK, resBody.IEKNonce, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open IEK: %w", err)
	}
	return generatedDEKs, iek, nil
}

func (ess *EnclaveSecureSession) GetTableIEK(ctx context.Context, tableHash string) ([]byte, error) {
	if time.Now().After(ess.ExpiresAt) {
		return nil, errors.New("decrypt session has expired")
	}
	if !ess.AttestationVerified {
		return nil, errors.New("attestation not verified")
	}

	body := encryption.SessionGetTableIEKRequest{
		SessionId: ess.SessionId,
		TableHash: tableHash,
	}

	reqBytes, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ess.endpoint+"/secure-session/get-table-iek", strings.NewReader(string(reqBytes)))
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
		var errBody errBody
		if err := json.NewDecoder(res.Body).Decode(&errBody); err == nil {
			bodyBytes, _ := io.ReadAll(res.Body)
			return nil, fmt.Errorf("failed to get table IEK: status %d, error: %s", res.StatusCode, string(bodyBytes))
		}
		return nil, fmt.Errorf("failed to get table IEK: status %d", res.StatusCode)
	}
	var resBody encryption.SessionGetTableIEKResponse
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, err
	}
	iek, err := openDEK(ess.SessionKey, resBody.SealedIEK, resBody.IEKNonce, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open IEK: %w", err)
	}
	return iek, nil
}

func (ess *EnclaveSecureSession) UnsealDEK(ctx context.Context, encryptedDEK []byte, nonce []byte, objectID string) ([]byte, error) {
	if ess.SessionKey == nil || len(ess.SessionKey) != chacha20poly1305.KeySize {
		return nil, errors.New("invalid session key")
	}

	dek, err := openDEK(ess.SessionKey, encryptedDEK, nonce, []byte(objectID))
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

func UnwrapSingleDEK(ctx context.Context, endpoint string, tenantID string, apiKey string, wrappedDEK []byte, nonce []byte) ([]byte, error) {
	clientPub, clientPriv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	body := encryption.DecryptRequest{
		Ciphertext:               wrappedDEK,
		Nonce:                    nonce,
		ClientEphemeralPublicKey: clientPub[:],
	}

	reqBytes, _ := json.Marshal(body)
	httpClient := &http.Client{Timeout: 15 * time.Second, Transport: tenantRoundTripper{TenantID: tenantID, APIKey: apiKey}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/decrypt", strings.NewReader(string(reqBytes)))
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
		var errBody errBody
		if err := json.NewDecoder(res.Body).Decode(&errBody); err == nil {
			bodyBytes, _ := io.ReadAll(res.Body)
			return nil, fmt.Errorf("failed to start decrypt session: status %d, error: %s", res.StatusCode, string(bodyBytes))
		}
		return nil, fmt.Errorf("failed to start decrypt session: status %d", res.StatusCode)
	}
	var resBody encryption.DecryptResponse
	if err := json.NewDecoder(res.Body).Decode(&resBody); err != nil {
		return nil, err
	}
	if len(resBody.EnclavePubKey) != 32 {
		return nil, errors.New("invalid enclave public key length")
	}
	var attNonce [24]byte
	copy(attNonce[:], resBody.Ciphertext[:24])

	// TODO: verify attestation in resBody.Attestation

	dek, ok := box.Open(nil, resBody.Ciphertext[24:], &attNonce, (*[32]byte)(resBody.EnclavePubKey), clientPriv)
	if !ok {
		return nil, errors.New("failed to decrypt DEK with NaCl box")
	}
	return dek, nil
}
