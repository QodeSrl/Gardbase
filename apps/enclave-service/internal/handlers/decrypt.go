package handlers

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"

	"github.com/qodesrl/gardbase/pkg/enclaveproto"
	"golang.org/x/crypto/nacl/box"

	"github.com/qodesrl/gardbase/apps/enclave-service/internal/utils"
)

func HandleDecrypt(encoder *json.Encoder, payload json.RawMessage, nsmPrivKey *rsa.PrivateKey) {
	var req enclaveproto.DecryptRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid decrypt request: %v", err))
		return
	}

	if len(req.ClientEphemeralPublicKey) != 32 {
		utils.SendError(encoder, "Invalid client ephemeral key")
		return
	}
	var clientPubKey [32]byte
	copy(clientPubKey[:], req.ClientEphemeralPublicKey)

	// decrypt the ciphertext for recipient using NSM private key
	decryptedOutput, err := utils.DecryptWithOpenSSL(req.Ciphertext, nsmPrivKey)
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

	resp := enclaveproto.Response[enclaveproto.DecryptResponse]{
		Success: true,
		Message: "Decryption successful",
		Data: enclaveproto.DecryptResponse{
			EnclavePubKey: enclavePubKey[:],
			Ciphertext:    ciphertextBox,
			Nonce:         nonce[:],
			RequestNonce:  req.Nonce,
		},
	}

	utils.SendResponse(encoder, resp)
}
