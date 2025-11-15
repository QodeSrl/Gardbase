package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/hf/nsm"
	"github.com/hf/nsm/request"
)

type AttestationRequest struct {
	Nonce     string `json:"nonce"`
	UserData  string `json:"user_data,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
}

func HandleAttestation(encoder *json.Encoder, payload json.RawMessage, nsmSession *nsm.Session)  {
	if nsmSession == nil {
		utils.SendError(encoder, "NSM not available (not running in enclave)")
		return
	}
	var req AttestationRequest
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &req); err != nil {
			utils.SendError(encoder, fmt.Sprintf("Invalid attestation request: %v", err))
			return
		}
	}

	attestationReq := request.Attestation{}

	if req.Nonce != "" {
		nonce, err := base64.StdEncoding.DecodeString(req.Nonce)
		if err != nil {
			utils.SendError(encoder, fmt.Sprintf("Invalid nonce encoding: %v", err))
			return
		}
		attestationReq.Nonce = nonce
	}

	if req.UserData != "" {
		userData, err := base64.StdEncoding.DecodeString(req.UserData)
		if err != nil {
			utils.SendError(encoder, fmt.Sprintf("Invalid user data encoding: %v", err))
			return
		}
		attestationReq.UserData = userData
	}

	if req.PublicKey != "" {
		publicKey, err := base64.StdEncoding.DecodeString(req.PublicKey)
		if err != nil {
			utils.SendError(encoder, fmt.Sprintf("Invalid user data encoding: %v", err))
			return
		}
		attestationReq.PublicKey = publicKey
	}

	// generate attestation document
	res, err := nsmSession.Send(&attestationReq)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to generate attestation document: %v", err))
		return
	}
	if res.Attestation == nil || res.Attestation.Document == nil {
		utils.SendError(encoder, "Attestation document is nil")
		return
	}

	response := utils.Response{
		Success: true,
		Message: "Attestation generated",
		Data: map[string]string{
			"attestation_document": base64.StdEncoding.EncodeToString(res.Attestation.Document),
			"timestamp": time.Now().String(),
		},
	}

	utils.SendResponse(encoder, response)
}