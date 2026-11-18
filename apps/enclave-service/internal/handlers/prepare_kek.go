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

func HandlePrepareKEK(payload json.RawMessage, nsmSession *nsm.Session, nsmPrivKey *rsa.PrivateKey) (res enclaveproto.PrepareKEKResponse, err error) {
	var req enclaveproto.PrepareKEKRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return res, fmt.Errorf("failed to unmarshal PrepareKEKRequest: %v", err)
	}

	if len(req.ClientEphemeralPublicKey) != 32 {
		return res, fmt.Errorf("invalid client ephemeral key length: %d", len(req.ClientEphemeralPublicKey))
	}
	var clientPubKey [32]byte
	copy(clientPubKey[:], req.ClientEphemeralPublicKey)

	var nonce [24]byte
	if _, err := nsmSession.Read(nonce[:]); err != nil {
		return res, fmt.Errorf("failed to read nonce from NSM: %v", err)
	}

	// decrypt the master key for recipient using NSM private key
	masterKey, err := utils.DecryptWithOpenSSL(req.MasterKey, nsmPrivKey)
	if err != nil {
		return res, fmt.Errorf("failed to decrypt master key for recipient: %v", err)
	}
	// decrypt the table salt for recipient using NSM private key
	tableSalt, err := utils.DecryptWithOpenSSL(req.TableSalt, nsmPrivKey)
	if err != nil {
		return res, fmt.Errorf("failed to decrypt table salt for recipient: %v", err)
	}

	// encrypt the decrypted DEK using NaCl box with client's ephemeral public key
	enclavePubKey, enclavePrivKey, err := box.GenerateKey(nsmSession)
	if err != nil {
		return res, fmt.Errorf("failed to generate enclave key pair: %v", err)
	}

	encryptedMasterKey := box.Seal(nonce[:], masterKey, &nonce, &clientPubKey, enclavePrivKey)
	encryptedTableSalt := box.Seal(nonce[:], tableSalt, &nonce, &clientPubKey, enclavePrivKey)

	res = enclaveproto.PrepareKEKResponse{
		EnclavePubKey: enclavePubKey[:],
		MasterKey:     encryptedMasterKey,
		TableSalt:     encryptedTableSalt,
		Nonce:         nonce[:],
	}

	return res, nil
}
