package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/session"
	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/hf/nsm"
)

type SessionUnwrapRequest struct {
	// Session ID, Base64-encoded
	SessionId string `json:"session_id,omitempty"`
	// KMS Key ID
	KeyId string `json:"key_id,omitempty"`
	Items     []struct {
		// Object ID
		ObjectId string `json:"object_id"`
		// Encrypted DEK, Base64-encoded
		Ciphertext string `json:"ciphertext"`
	} `json:"items"`
}

type sessionUnwrapItemResult struct {
	// Object ID
	ObjectId string `json:"object_id"`
	// Decrypted DEK, Base64-encoded
	Ciphertext string `json:"ciphertext"`
	// Nonce used for decryption, Base64-encoded
	Nonce   string `json:"nonce"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type SessionUnwrapResponse []sessionUnwrapItemResult

func HandleSessionUnwrap(encoder *json.Encoder, payload json.RawMessage, nsmSession *nsm.Session, nsmPrivKey *rsa.PrivateKey, kmsClient *kms.Client) {
	var req SessionUnwrapRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid session unwrap request: %v", err))
		return
	}

	sess, ok := session.GetSession(req.SessionId)
	if !ok {
		utils.SendError(encoder, "Invalid or expired session ID")
		return
	}

	aead, err := chacha20poly1305.NewX(sess.Key)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to create AEAD cipher: %v", err))
		return
	}

	results := make(SessionUnwrapResponse, 0, len(req.Items))
	for _, it := range req.Items {
		objId := it.ObjectId
		if objId == "" {
			results = append(results, sessionUnwrapItemResult{
				ObjectId:  objId,
				Success: false,
				Error: "missing object_id",
			})
			continue
		}
		if it.Ciphertext == "" {
			results = append(results, sessionUnwrapItemResult{
				ObjectId:  objId,
				Success: false,
				Error: "missing ciphertext",
			})
			continue
		}
		ctBytes, err := base64.StdEncoding.DecodeString(it.Ciphertext)
		if err != nil {
			results = append(results, sessionUnwrapItemResult{
				ObjectId:  objId,
				Success: false,
				Error: fmt.Sprintf("invalid base64 ciphertext: %v", err),
			})
			continue
		}

		ctx := context.Background()
		input := &kms.DecryptInput{
			CiphertextBlob: ctBytes,
			KeyId: &req.KeyId,
			Recipient: &kmsTypes.RecipientInfo{
				// client has already verified the attestation document when establishing the session
				AttestationDocument: nil,
				KeyEncryptionAlgorithm: "RSAES_OAEP_SHA_256",
			},
		}
		output, err := kmsClient.Decrypt(ctx, input)
		if err != nil {
			results = append(results, sessionUnwrapItemResult{
				ObjectId:  objId,
				Success: false,
				Error: fmt.Sprintf("KMS decrypt failed: %v", err),
			})
			continue
		}
		plainDEK, err := rsa.DecryptOAEP(sha256.New(), nsmSession, nsmPrivKey, output.CiphertextForRecipient, nil)
		if err != nil {
			results = append(results, sessionUnwrapItemResult{
				ObjectId:  objId,
				Success: false,
				Error: fmt.Sprintf("RSA decryption failed: %v", err),
			})
			continue
		}

		nonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
		if _, err := rand.Read(nonce); err != nil {
			results = append(results, sessionUnwrapItemResult{
				ObjectId: objId,
				Success:  false,
				Error:    fmt.Sprintf("Failed to generate nonce: %v", err),
			})
			continue
		}

		ciphertextBox := aead.Seal(nil, nonce, plainDEK, []byte(objId))

		results = append(results, sessionUnwrapItemResult{
			ObjectId:  objId,
			Ciphertext: base64.StdEncoding.EncodeToString(ciphertextBox),
			Nonce:      base64.StdEncoding.EncodeToString(nonce),
			Success:    true,
		})
	}

	utils.SendResponse(encoder, utils.Response{
		Success: true,
		Data: results,
	})
}