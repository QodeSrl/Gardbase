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

func HandleSessionUnwrap(encoder *json.Encoder, payload json.RawMessage, nsmSession *nsm.Session, nsmPrivKey *rsa.PrivateKey) {
	var req enclaveproto.SessionUnwrapRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid session unwrap request: %v", err))
		return
	}

	sess, ok := session.GetSession(req.SessionId)
	if !ok {
		utils.SendError(encoder, "Invalid or expired session ID")
		return
	}

	aead, err := chacha20poly1305.NewX(sess.Key)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to create AEAD cipher: %v", err))
		return
	}

	results := make([]enclaveproto.SessionUnwrapItemResult, 0, len(req.Items))

	for _, it := range req.Items {
		objId := it.ObjectId
		// note: here nsmSession is used as a rand.Reader
		plainDEK, err := utils.DecryptWithOpenSSL(it.Ciphertext, nsmPrivKey)
		if err != nil {
			results = append(results, enclaveproto.SessionUnwrapItemResult{
				ObjectId: objId,
				Success:  false,
				Error:    fmt.Sprintf("RSA decryption failed: %v", err),
			})
			continue
		}

		// nonce for session encryption
		nonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
		if _, err := nsmSession.Read(nonce); err != nil {
			results = append(results, enclaveproto.SessionUnwrapItemResult{
				ObjectId: objId,
				Success:  false,
				Error:    fmt.Sprintf("Failed to generate nonce: %v", err),
			})
			continue
		}

		// seal DEK with session key using object ID as associated data
		sealedDEK := aead.Seal(nil, nonce, plainDEK, []byte(objId))

		utils.Zero(plainDEK)

		results = append(results, enclaveproto.SessionUnwrapItemResult{
			ObjectId:  objId,
			SealedDEK: sealedDEK,
			Nonce:     nonce,
			Success:   true,
		})
	}

	utils.SendResponse(encoder, enclaveproto.Response[enclaveproto.SessionUnwrapResponse]{
		Success: true,
		Data:    results,
	})
}
