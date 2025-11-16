package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/hf/nsm"
	"github.com/hf/nsm/request"
	"golang.org/x/crypto/nacl/box"
)

type DecryptRequest struct {
	Ciphertext string `json:"ciphertext"` // Base64-encoded (encrypted dek)
	Nonce      string `json:"nonce,omitempty"`
	KeyID 	   string `json:"key_id,omitempty"`
}

type DecryptResponse struct {
	EnclavePubKey string `json:"enclave_public_key"` // Base64-encoded (x25519 pub key)
	Ciphertext    string `json:"ciphertext"`         // Base64-encoded 
	Nonce         string `json:"nonce"`              // Base64-encoded
	RequestNonce  string `json:"request_nonce,omitempty"`
}

func HandleDecrypt(encoder *json.Encoder, payload json.RawMessage, nsmSession *nsm.Session, kmsClient *kms.Client, clientEphemeralPublicKey string, pubKeyBytes []byte, privKey *rsa.PrivateKey) {
	clientPubKeyBytes, err := base64.StdEncoding.DecodeString(clientEphemeralPublicKey);
	if err != nil || len(clientPubKeyBytes) != 32 {
		utils.SendError(encoder, "Invalid client ephemeral key")
		return
	}
	var clientPubKey [32]byte
	copy(clientPubKey[:], clientPubKeyBytes)

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
		attestationReq := request.Attestation{
			PublicKey: pubKeyBytes,
		}
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
		KeyId: &req.KeyID,
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

	if len(output.CiphertextForRecipient) == 0 {
    	utils.SendError(encoder, "KMS did not return ciphertext for recipient")
    	return
	}

	decryptedOutput, err := rsa.DecryptOAEP(
		sha256.New(),
		nsmSession,
		privKey,
		output.CiphertextForRecipient,
		nil,
	)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to decrypt ciphertext for recipient: %v", err))
		return
	}

	var enclavePubKey, enclavePrivKey *[32]byte
	enclavePubKey, enclavePrivKey, err = box.GenerateKey(rand.Reader)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to generate enclave key pair: %v", err))
		return
	}

	nonce := new([24]byte)
	if _, err := rand.Read(nonce[:]); err != nil {
		utils.SendError(encoder, "Failed to generate nonce")
		return
	}

	ciphertextBox := box.Seal(nonce[:], decryptedOutput, nonce, &clientPubKey, enclavePrivKey)

	resp := utils.Response{
		Success: true,
		Message: "Decryption successful",
		Data: DecryptResponse{
			EnclavePubKey: base64.StdEncoding.EncodeToString(enclavePubKey[:]),
			Ciphertext:    base64.StdEncoding.EncodeToString(ciphertextBox),
			Nonce:         base64.StdEncoding.EncodeToString(nonce[:]),
			RequestNonce:  req.Nonce,
		},
	}

	utils.SendResponse(encoder, resp)
}
