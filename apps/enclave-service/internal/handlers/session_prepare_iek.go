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

	iek, err := utils.DecryptWithOpenSSL(req.IEK, nsmPrivKey)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to decrypt IEK: %v", err))
		return
	}

	sealedIEK := sessAead.Seal(nil, sessNonce, iek, nil)

	utils.Zero(iek)

	res := enclaveproto.PrepareIEKResponse{
		SealedIEK: sealedIEK,
		IEKNonce:  sessNonce,
	}

	utils.SendResponse(encoder, enclaveproto.Response[enclaveproto.PrepareIEKResponse]{
		Success: true,
		Data:    res,
	})
}
