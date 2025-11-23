package crypto

import (
	"context"
"crypto/rand"
	"encoding/base64"
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
	AttestationVerified bool
	endpoint            string // base URL of proxy
	httpClient          *http.Client
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
func StartDecryptSession(ctx context.Context, endpoint string, clientPriv [32]byte, clientPub [32]byte, clientPubB64 string, nonceB64 string) (*DecryptSession, error) {
	reqBody := enclaveproto.SessionInitRequest{
		ClientEphemeralPublicKey: clientPubB64,
		Nonce:                    nonceB64,
	}
	reqBytes, _ := json.Marshal(reqBody)
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
		endpoint:            endpoint,
		httpClient:          httpClient,
		AttestationVerified: false,
	}
	return ds, nil
}

func (ds *DecryptSession) SessionUnwrap(ctx context.Context, items []enclaveproto.SessionUnwrapItem, keyID string) (enclaveproto.SessionUnwrapResponse, error) {
	if time.Now().After(ds.ExpiresAt) {
		return nil, errors.New("decrypt session has expired")
	}
	if !ds.AttestationVerified {
		if !verifyAttestation(ds) {
			return nil, errors.New("attestation verification failed")
		}
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
