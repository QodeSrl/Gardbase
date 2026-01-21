package handlers

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/session"
	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/hf/nsm"
	"golang.org/x/crypto/chacha20poly1305"
)

func HandleSessionPrepareDEK(encoder *json.Encoder, payload json.RawMessage, nsmSession *nsm.Session, nsmPrivKey *rsa.PrivateKey) {
	var req enclaveproto.EnclavePrepareDEKRequest
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

	wrappedMasterKey, err := base64.StdEncoding.DecodeString(req.WrappedMasterKey)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to decode wrapped master key: %v", err))
		return
	}
	masterKey, err := utils.DecryptWithOpenSSL(wrappedMasterKey, nsmPrivKey)
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
		ciphertextForRecipient, err := base64.StdEncoding.DecodeString(dek.CiphertextForRecipient)
		if err != nil {
			utils.SendError(encoder, fmt.Sprintf("Failed to decode CiphertextForRecipient: %v", err))
			return
		}
		plainDEK, err := utils.DecryptWithOpenSSL(ciphertextForRecipient, nsmPrivKey)
		if err != nil {
			utils.SendError(encoder, fmt.Sprintf("Failed to decrypt data key: %v", err))
			return
		}

		// seal DEK with session key
		sealedDEK := sessAead.Seal(nil, sessNonce, plainDEK, nil)

		// encrypt DEK with master key
		masterEncryptedDEK := masterKeyAead.Seal(nil, masterKeyNonce, plainDEK, nil)

		utils.Zero(plainDEK)

		results = append(results, enclaveproto.GeneratedDEK{
			SealedDEK:          base64.StdEncoding.EncodeToString(sealedDEK),
			KmsEncryptedDEK:    dek.CiphertextBlob,
			MasterEncryptedDEK: base64.StdEncoding.EncodeToString(masterEncryptedDEK),
			SessionNonce:       base64.StdEncoding.EncodeToString(sessNonce),
			MasterKeyNonce:     base64.StdEncoding.EncodeToString(masterKeyNonce),
		})
	}

	utils.Zero(masterKey)

	res := enclaveproto.SessionGenerateDEKResponse{
		DEKs: results,
	}

	utils.SendResponse(encoder, enclaveproto.Response[enclaveproto.SessionGenerateDEKResponse]{
		Success: true,
		Data:    res,
	})
}
