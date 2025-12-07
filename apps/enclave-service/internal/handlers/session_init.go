package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/session"
	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/hf/nsm"
	"github.com/hf/nsm/request"
	"golang.org/x/crypto/curve25519"
)

func HandleSessionInit(encoder *json.Encoder, payload json.RawMessage, nsmSession *nsm.Session) {
	var req enclaveproto.SessionInitRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid decrypt request: %v", err))
		return
	}

	// decode client ephemeral public key
	clientPubKeyBytes, err := base64.StdEncoding.DecodeString(req.ClientEphemeralPublicKey)
	if err != nil || len(clientPubKeyBytes) != 32 {
		utils.SendError(encoder, "Invalid client ephemeral key")
		return
	}
	var clientPubKey [32]byte
	copy(clientPubKey[:], clientPubKeyBytes)

	// generate enclave ephemeral keypair
	var ephPriv, ephPub [32]byte
	if _, err := rand.Read(ephPriv[:]); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to generate enclave ephemeral key: %v", err))
		return
	}
	curve25519.ScalarBaseMult(&ephPub, &ephPriv)

	// derive session key (shared secret, x25519 + hkdf)
	sessKey, err := utils.DeriveSessionKey(ephPriv, clientPubKeyBytes)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to derive session key: %v", err))
		return
	}

	// generate session ID
	sid := make([]byte, 16)
	if _, err := rand.Read(sid); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to generate session ID: %v", err))
		return
	}
	sidB64 := base64.StdEncoding.EncodeToString(sid)

	// ttl
	ttl := 60 * time.Minute
	expiresAt := time.Now().Add(ttl)

	// store session in enclave memory
	session.StoreSession(sidB64, sessKey, expiresAt)

	// request attestation doc (generate a new one with session pubkey + client's nonce)
	var sessionAttDoc []byte
	if nsmSession != nil {
		attestationReq := request.Attestation{
			PublicKey: ephPub[:],
		}
		if req.Nonce != "" {
			nonce, err := base64.StdEncoding.DecodeString(req.Nonce)
			if err != nil {
				utils.SendError(encoder, fmt.Sprintf("Invalid nonce encoding: %v", err))
				return
			}
			attestationReq.Nonce = nonce
		}
		attestationRes, err := nsmSession.Send(&attestationReq)
		if err != nil {
			utils.SendError(encoder, fmt.Sprintf("Failed to get attestation document: %v", err))
			return
		}
		if attestationRes.Attestation != nil && attestationRes.Attestation.Document != nil {
			sessionAttDoc = attestationRes.Attestation.Document
		}
	}

	defer utils.Zero(ephPriv[:])

	res := enclaveproto.SessionInitResponse{
		SessionId:                 sidB64,
		EnclaveEphemeralPublicKey: base64.StdEncoding.EncodeToString(ephPub[:]),
		ExpiresAt:                 expiresAt.Format(time.RFC3339),
	}
	if len(sessionAttDoc) > 0 {
		res.Attestation = base64.StdEncoding.EncodeToString(sessionAttDoc)
	}
	utils.SendResponse(encoder, enclaveproto.Response[enclaveproto.SessionInitResponse]{Success: true, Data: res})
}
