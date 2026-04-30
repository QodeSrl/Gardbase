package handlers

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/session"
	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/hf/nsm"
	"golang.org/x/crypto/chacha20poly1305"
)

func HandleSessionPrepareDEK(encoder *json.Encoder, payload json.RawMessage, nsmSession *nsm.Session, nsmPrivKey *rsa.PrivateKey) {
	var req enclaveproto.PrepareDEKRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid session generate DEK request: %v", err))
		return
	}

	sess, ok := session.GetSession(req.SessionId)
	if !ok {
		utils.SendError(encoder, "Invalid or expired session ID")
		return
	}

	sessAead, err := chacha20poly1305.NewX(sess.Key)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to create AEAD cipher: %v", err))
		return
	}

	sessNonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
	if _, err := nsmSession.Read(sessNonce); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to read session nonce: %v", err))
		return
	}

	masterKey, err := utils.DecryptWithOpenSSL(req.WrappedMasterKey, nsmPrivKey)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to decrypt master key: %v", err))
		return
	}

	masterKeyAead, err := chacha20poly1305.NewX(masterKey)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to create master key AEAD cipher: %v", err))
		return
	}

	masterKeyNonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
	if _, err := nsmSession.Read(masterKeyNonce); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to read master key nonce: %v", err))
		return
	}

	results := make([]enclaveproto.GeneratedDEK, 0, len(req.DEKs))

	for _, dek := range req.DEKs {
		plainDEK, err := utils.DecryptWithOpenSSL(dek.CiphertextForRecipient, nsmPrivKey)
		if err != nil {
			utils.SendError(encoder, fmt.Sprintf("Failed to decrypt data key: %v", err))
			return
		}

		dekSessNonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
		if _, err := nsmSession.Read(dekSessNonce); err != nil {
			utils.SendError(encoder, fmt.Sprintf("Failed to read DEK session nonce: %v", err))
			return
		}
		dekMasterNonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
		if _, err := nsmSession.Read(dekMasterNonce); err != nil {
			utils.SendError(encoder, fmt.Sprintf("Failed to read DEK master nonce: %v", err))
			return
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
		utils.SendError(encoder, fmt.Sprintf("Failed to decrypt IEK: %v", err))
		return
	}

	iekNonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
	if _, err := nsmSession.Read(iekNonce); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to read IEK nonce: %v", err))
		return
	}

	sealedIEK := sessAead.Seal(nil, iekNonce, iek, nil)

	utils.Zero(iek)
	utils.Zero(masterKey)

	res := enclaveproto.PrepareDEKResponse{
		DEKs:      results,
		SealedIEK: sealedIEK,
		IEKNonce:  iekNonce,
	}

	utils.SendResponse(encoder, enclaveproto.Response[enclaveproto.PrepareDEKResponse]{
		Success: true,
		Data:    res,
	})
}
