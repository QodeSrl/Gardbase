package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/hf/nsm"
	"github.com/hf/nsm/request"
)

type DecryptRequest struct {
	Ciphertext string `json:"ciphertext"` // Base64-encoded (encrypted dek)
	Nonce      string `json:"nonce,omitempty"`
	TenantID   string `json:"tenant_id,omitempty"`
	ObjectID   string `json:"object_id,omitempty"`
}

type DecryptResponse struct {
	Plaintext string `json:"plaintext"` // Base64-encoded
}

func HandleDecrypt(encoder *json.Encoder, payload json.RawMessage, nsmSession *nsm.Session, kmsClient *kms.Client) {
	var req DecryptRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid decrypt request: %v", err))
		return
	}

	if req.Ciphertext == "" {
		utils.SendError(encoder, "Ciphertext is required")
		return
	}

	ciphertext, err := base64.StdEncoding.DecodeString(req.Ciphertext);
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid ciphertext encoding: %v", err))
		return
	}

	var attestationDoc []byte
	if nsmSession != nil {
		attestationReq := request.Attestation{}
		if req.Nonce != "" {
			nonce, err := base64.StdEncoding.DecodeString(req.Nonce)
			if err != nil {
				utils.SendError(encoder, fmt.Sprintf("Invalid nonce encoding: %v", err))
				return
			}
			attestationReq.Nonce = nonce
		}
		attestationRes, err := nsmSession.Send(&attestationReq)
		if err != nil {
			utils.SendError(encoder, fmt.Sprintf("Failed to get attestation document: %v", err))
			return
		}
		if attestationRes.Attestation != nil && attestationRes.Attestation.Document != nil {
			attestationDoc = attestationRes.Attestation.Document
		}
	}

	// call kms to decrypt
	ctx := context.Background()
	input := &kms.DecryptInput{
		CiphertextBlob: ciphertext,
	}

	if attestationDoc != nil {
		input.Recipient = &kmsTypes.RecipientInfo{
			AttestationDocument: attestationDoc,
			KeyEncryptionAlgorithm: "RSAES_OAEP_SHA_256",
		}
	}

	output, err := kmsClient.Decrypt(ctx, input)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("KMS decrypt failed: %v", err))
		return
	}

	resp := utils.Response{
		Success: true,
		Message: "Decryption successful",
		Data: DecryptResponse{
			Plaintext: base64.StdEncoding.EncodeToString(output.Plaintext),
		},
	}

	utils.SendResponse(encoder, resp)
}
