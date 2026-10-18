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

func HandleDecrypt(payload json.RawMessage, nsmPrivKey *rsa.PrivateKey) (res enclaveproto.DecryptResponse, err error) {
	var req enclaveproto.DecryptRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return res, fmt.Errorf("Invalid decrypt request: %v", err)
	}

	if len(req.ClientEphemeralPublicKey) != 32 {
		return res, fmt.Errorf("Invalid client ephemeral key length: %d", len(req.ClientEphemeralPublicKey))
	}
	var clientPubKey [32]byte
	copy(clientPubKey[:], req.ClientEphemeralPublicKey)

	// decrypt the ciphertext for recipient using NSM private key
	decryptedOutput, err := utils.DecryptWithOpenSSL(req.Ciphertext, nsmPrivKey)
	if err != nil {
		return res, fmt.Errorf("Failed to decrypt ciphertext for recipient: %v", err)
	}

	// encrypt the decrypted DEK using NaCl box with client's ephemeral public key
	enclavePubKey, enclavePrivKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return res, fmt.Errorf("Failed to generate enclave key pair: %v", err)
	}
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return res, fmt.Errorf("Failed to generate nonce: %v", err)
	}
	ciphertextBox := box.Seal(nonce[:], decryptedOutput, &nonce, &clientPubKey, enclavePrivKey)

	res = enclaveproto.DecryptResponse{
		EnclavePubKey: enclavePubKey[:],
		Ciphertext:    ciphertextBox,
		Nonce:         nonce[:],
		RequestNonce:  req.Nonce,
	}

	return res, nil
}
