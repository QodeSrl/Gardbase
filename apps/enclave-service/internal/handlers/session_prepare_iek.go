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

func HandleSessionPrepareIEK(payload json.RawMessage, nsmSession *nsm.Session, nsmPrivKey *rsa.PrivateKey) (res enclaveproto.PrepareIEKResponse, err error) {
	var req enclaveproto.PrepareIEKRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return res, fmt.Errorf("Invalid session prepare IEK request: %v", err)
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

	iek, err := utils.DecryptWithOpenSSL(req.IEK, nsmPrivKey)
	if err != nil {
		return res, fmt.Errorf("Failed to decrypt IEK: %v", err)
	}

	sealedIEK := sessAead.Seal(nil, sessNonce, iek, nil)

	utils.Zero(iek)

	res = enclaveproto.PrepareIEKResponse{
		SealedIEK: sealedIEK,
		IEKNonce:  sessNonce,
	}

	return res, nil
}
