package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hf/nsm"
	"github.com/hf/nsm/request"
	"github.com/qodesrl/gardbase/apps/enclave-service/internal/session"
	"github.com/qodesrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/qodesrl/gardbase/pkg/enclaveproto"
	"golang.org/x/crypto/curve25519"
)

func HandleSessionInit(payload json.RawMessage, nsmSession *nsm.Session) (res enclaveproto.SessionInitResponse, err error) {
	var req enclaveproto.SessionInitRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return res, fmt.Errorf("Invalid decrypt request: %v", err)
	}

	if len(req.ClientEphemeralPublicKey) != 32 {
		return res, fmt.Errorf("Invalid client ephemeral key length: %d", len(req.ClientEphemeralPublicKey))
	}
	var clientPubKey [32]byte
	copy(clientPubKey[:], req.ClientEphemeralPublicKey)

	// generate enclave ephemeral keypair
	var ephPriv, ephPub [32]byte
	if _, err := rand.Read(ephPriv[:]); err != nil {
		return res, fmt.Errorf("Failed to generate enclave ephemeral key: %v", err)
	}
	curve25519.ScalarBaseMult(&ephPub, &ephPriv)

	// derive session key (shared secret, x25519 + hkdf)
	sessKey, err := utils.DeriveSessionKey(ephPriv, req.ClientEphemeralPublicKey)
	if err != nil {
		return res, fmt.Errorf("Failed to derive session key: %v", err)
	}

	// generate session ID
	sid := make([]byte, 16)
	if _, err := rand.Read(sid); err != nil {
		return res, fmt.Errorf("Failed to generate session ID: %v", err)
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
		if req.Nonce != nil {
			attestationReq.Nonce = req.Nonce
		}
		attestationRes, err := nsmSession.Send(&attestationReq)
		if err != nil {
			return res, fmt.Errorf("Failed to get attestation document: %v", err)
		}
		if attestationRes.Attestation != nil && attestationRes.Attestation.Document != nil {
			sessionAttDoc = attestationRes.Attestation.Document
		}
	}

	defer utils.Zero(ephPriv[:])

	res = enclaveproto.SessionInitResponse{
		SessionId:                 sidB64,
		EnclaveEphemeralPublicKey: ephPub[:],
		ExpiresAt:                 expiresAt.Format(time.RFC3339),
	}
	if len(sessionAttDoc) > 0 {
		res.Attestation = sessionAttDoc
	}
	return res, nil
}
