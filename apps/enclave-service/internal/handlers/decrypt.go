package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/QodeSrl/gardbase/pkg/enclaveproto"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/hf/nsm"
	"github.com/hf/nsm/request"
	"golang.org/x/crypto/nacl/box"
)

func HandleDecrypt(encoder *json.Encoder, payload json.RawMessage, nsmSession *nsm.Session, kmsClient *kms.Client, pubKeyBytes []byte, nsmPrivKey *rsa.PrivateKey) {
	var req enclaveproto.DecryptRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid decrypt request: %v", err))
		return
	}

	clientPubKeyBytes, err := base64.StdEncoding.DecodeString(req.ClientEphemeralPublicKey);
	if err != nil || len(clientPubKeyBytes) != 32 {
		utils.SendError(encoder, "Invalid client ephemeral key")
		return
	}
	var clientPubKey [32]byte
	copy(clientPubKey[:], clientPubKeyBytes)

	ciphertext, err := base64.StdEncoding.DecodeString(req.Ciphertext);
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid ciphertext encoding: %v", err))
		return
	}

	// generate attestation document for KMS, based on NSM public key and (request's) nonce
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

	// include attestation document if available
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

	// decrypt the ciphertext for recipient using NSM private key
	decryptedOutput, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, nsmPrivKey, output.CiphertextForRecipient, nil)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to decrypt ciphertext for recipient: %v", err))
		return
	}

	// encrypt the decrypted DEK using NaCl box with client's ephemeral public key
	enclavePubKey, enclavePrivKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to generate enclave key pair: %v", err))
		return
	}
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		utils.SendError(encoder, "Failed to generate nonce")
		return
	}
	ciphertextBox := box.Seal(nonce[:], decryptedOutput, &nonce, &clientPubKey, enclavePrivKey)

	// On the client side, the DEK can be decrypted using:
	// box.Open(nil, ciphertextBox, &nonce, &enclavePubKey, clientPrivKey)

	resp := enclaveproto.Response{
		Success: true,
		Message: "Decryption successful",
		Data: enclaveproto.DecryptResponse{
			EnclavePubKey: base64.StdEncoding.EncodeToString(enclavePubKey[:]),
			Ciphertext:    base64.StdEncoding.EncodeToString(ciphertextBox),
			Nonce:         base64.StdEncoding.EncodeToString(nonce[:]),
			RequestNonce:  req.Nonce,
			Attestation: base64.StdEncoding.EncodeToString(attestationDoc),
		},
	}

	utils.SendResponse(encoder, resp)
}
