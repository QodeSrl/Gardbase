package handlers

import (
	"encoding/base64"
	"encoding/json"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
)

func HandleGetAttestation(encoder *json.Encoder, nsmAttestation []byte) {
	attestationB64 := base64.StdEncoding.EncodeToString(nsmAttestation)
	res := enclaveproto.GetAttestationResponse{
		Attestation: attestationB64,
	}
	utils.SendResponse(encoder, enclaveproto.Response[enclaveproto.GetAttestationResponse]{
		Success: true,
		Data:    res,
	})
}
