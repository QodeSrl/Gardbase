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

func HandleSessionPrepareIEK(encoder *json.Encoder, payload json.RawMessage, nsmSession *nsm.Session, nsmPrivKey *rsa.PrivateKey) {
	var req enclaveproto.PrepareIEKRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid session prepare IEK request: %v", err))
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

	ciphertextForRecipient, err := base64.StdEncoding.DecodeString(req.IEK)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to decode IEK: %v", err))
		return
	}
	iek, err := utils.DecryptWithOpenSSL(ciphertextForRecipient, nsmPrivKey)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to decrypt IEK: %v", err))
		return
	}

	sealedIEK := sessAead.Seal(nil, sessNonce, iek, nil)

	utils.Zero(iek)

	res := enclaveproto.PrepareIEKResponse{
		SealedIEK: base64.StdEncoding.EncodeToString(sealedIEK),
		IEKNonce:  base64.StdEncoding.EncodeToString(sessNonce),
	}

	utils.SendResponse(encoder, enclaveproto.Response[enclaveproto.PrepareIEKResponse]{
		Success: true,
		Data:    res,
	})
}
