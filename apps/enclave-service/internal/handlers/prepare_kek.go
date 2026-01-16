package handlers

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/hf/nsm"
	"golang.org/x/crypto/nacl/box"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
)

func HandlePrepareKEK(encoder *json.Encoder, payload json.RawMessage, nsmPrivKey *rsa.PrivateKey, nsmSession *nsm.Session) {
	var req enclaveproto.EnclavePrepareKEKRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid decrypt request: %v", err))
		return
	}

	clientPubKeyBytes, err := base64.StdEncoding.DecodeString(req.ClientEphemeralPublicKey)
	if err != nil || len(clientPubKeyBytes) != 32 {
		utils.SendError(encoder, "Invalid client ephemeral key")
		return
	}
	var clientPubKey [32]byte
	copy(clientPubKey[:], clientPubKeyBytes)

	var nonce [24]byte
	if _, err := nsmSession.Read(nonce[:]); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to read nonce from NSM: %v", err))
		return
	}

	wrappedMasterKey, err := base64.StdEncoding.DecodeString(req.MasterKey)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid master key encoding: %v", err))
		return
	}

	wrappedTableSalt, err := base64.StdEncoding.DecodeString(req.TableSalt)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid table salt encoding: %v", err))
		return
	}

	// decrypt the master key for recipient using NSM private key
	masterKey, err := utils.DecryptWithOpenSSL(wrappedMasterKey, nsmPrivKey)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to decrypt ciphertext for recipient: %v", err))
		return
	}
	// decrypt the table salt for recipient using NSM private key
	tableSalt, err := utils.DecryptWithOpenSSL(wrappedTableSalt, nsmPrivKey)
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

	resp := enclaveproto.Response[enclaveproto.EnclavePrepareKEKResponse]{
		Success: true,
		Data: enclaveproto.EnclavePrepareKEKResponse{
			EnclavePubKey: base64.StdEncoding.EncodeToString(enclavePubKey[:]),
			MasterKey:     base64.StdEncoding.EncodeToString(encryptedMasterKey),
			TableSalt:     base64.StdEncoding.EncodeToString(encryptedTableSalt),
			Nonce:         base64.StdEncoding.EncodeToString(nonce[:]),
		},
	}

	utils.SendResponse(encoder, resp)
}
