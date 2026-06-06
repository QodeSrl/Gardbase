package handlers

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"

	"github.com/hf/nsm"
	"github.com/qodesrl/gardbase/pkg/enclaveproto"
	"golang.org/x/crypto/nacl/box"

	"github.com/qodesrl/gardbase/apps/enclave-service/internal/utils"
)

func HandlePrepareKEK(encoder *json.Encoder, payload json.RawMessage, nsmSession *nsm.Session, nsmPrivKey *rsa.PrivateKey) {
	var req enclaveproto.PrepareKEKRequest
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

	var nonce [24]byte
	if _, err := nsmSession.Read(nonce[:]); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to read nonce from NSM: %v", err))
		return
	}

	// decrypt the master key for recipient using NSM private key
	masterKey, err := utils.DecryptWithOpenSSL(req.MasterKey, nsmPrivKey)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to decrypt ciphertext for recipient: %v", err))
		return
	}
	// decrypt the table salt for recipient using NSM private key
	tableSalt, err := utils.DecryptWithOpenSSL(req.TableSalt, nsmPrivKey)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to decrypt table salt for recipient: %v", err))
		return
	}

	// encrypt the decrypted DEK using NaCl box with client's ephemeral public key
	enclavePubKey, enclavePrivKey, err := box.GenerateKey(nsmSession)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to generate enclave key pair: %v", err))
		return
	}

	encryptedMasterKey := box.Seal(nonce[:], masterKey, &nonce, &clientPubKey, enclavePrivKey)
	encryptedTableSalt := box.Seal(nonce[:], tableSalt, &nonce, &clientPubKey, enclavePrivKey)

	resp := enclaveproto.Response[enclaveproto.PrepareKEKResponse]{
		Success: true,
		Data: enclaveproto.PrepareKEKResponse{
			EnclavePubKey: enclavePubKey[:],
			MasterKey:     encryptedMasterKey,
			TableSalt:     encryptedTableSalt,
			Nonce:         nonce[:],
		},
	}

	utils.SendResponse(encoder, resp)
}
