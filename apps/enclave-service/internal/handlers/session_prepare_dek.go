package handlers

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"

	"github.com/hf/nsm"
	"github.com/qodesrl/gardbase/apps/enclave-service/internal/session"
	"github.com/qodesrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/qodesrl/gardbase/pkg/enclaveproto"
	"golang.org/x/crypto/chacha20poly1305"
)

func HandleSessionPrepareDEK(payload json.RawMessage, nsmSession *nsm.Session, nsmPrivKey *rsa.PrivateKey) (res enclaveproto.PrepareDEKResponse, err error) {
	var req enclaveproto.PrepareDEKRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return res, fmt.Errorf("Invalid session generate DEK request: %v", err)
	}

	sess, ok := session.GetSession(req.SessionId)
	if !ok {
		return res, fmt.Errorf("Invalid or expired session ID")
	}

	sessAead, err := chacha20poly1305.NewX(sess.Key)
	if err != nil {
		return res, fmt.Errorf("Failed to create AEAD cipher: %v", err)
	}

	sessNonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
	if _, err := nsmSession.Read(sessNonce); err != nil {
		return res, fmt.Errorf("Failed to read session nonce: %v", err)
	}

	masterKey, err := utils.DecryptWithOpenSSL(req.WrappedMasterKey, nsmPrivKey)
	if err != nil {
		return res, fmt.Errorf("Failed to decrypt master key: %v", err)
	}

	masterKeyAead, err := chacha20poly1305.NewX(masterKey)
	if err != nil {
		return res, fmt.Errorf("Failed to create master key AEAD cipher: %v", err)
	}

	masterKeyNonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
	if _, err := nsmSession.Read(masterKeyNonce); err != nil {
		return res, fmt.Errorf("Failed to read master key nonce: %v", err)
	}

	results := make([]enclaveproto.GeneratedDEK, 0, len(req.DEKs))

	for _, dek := range req.DEKs {
		plainDEK, err := utils.DecryptWithOpenSSL(dek.CiphertextForRecipient, nsmPrivKey)
		if err != nil {
			return res, fmt.Errorf("Failed to decrypt data key: %v", err)
		}

		dekSessNonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
		if _, err := nsmSession.Read(dekSessNonce); err != nil {
			return res, fmt.Errorf("Failed to read DEK session nonce: %v", err)
		}
		dekMasterNonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
		if _, err := nsmSession.Read(dekMasterNonce); err != nil {
			return res, fmt.Errorf("Failed to read DEK master nonce: %v", err)
		}

		// seal DEK with session key
		sealedDEK := sessAead.Seal(nil, dekSessNonce, plainDEK, nil)

		// encrypt DEK with master key
		masterEncryptedDEK := masterKeyAead.Seal(nil, dekMasterNonce, plainDEK, nil)

		utils.Zero(plainDEK)

		results = append(results, enclaveproto.GeneratedDEK{
			SealedDEK:          sealedDEK,
			KmsEncryptedDEK:    dek.CiphertextBlob,
			MasterEncryptedDEK: masterEncryptedDEK,
			SessionNonce:       dekSessNonce,
			MasterKeyNonce:     dekMasterNonce,
		})
	}

	iek, err := utils.DecryptWithOpenSSL(req.IEK, nsmPrivKey)
	if err != nil {
		return res, fmt.Errorf("Failed to decrypt IEK: %v", err)
	}

	iekNonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
	if _, err := nsmSession.Read(iekNonce); err != nil {
		return res, fmt.Errorf("Failed to read IEK nonce: %v", err)
	}

	sealedIEK := sessAead.Seal(nil, iekNonce, iek, nil)

	utils.Zero(iek)
	utils.Zero(masterKey)

	res = enclaveproto.PrepareDEKResponse{
		DEKs:      results,
		SealedIEK: sealedIEK,
		IEKNonce:  iekNonce,
	}

	return res, nil
}
