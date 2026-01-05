package handlers

import (
	"encoding/base64"
	"encoding/json"
	"log"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
)

func HandleGetAttestation(encoder *json.Encoder, attestation *utils.Attestation) {
	attestation.Mu.RLock()
	if len(attestation.Doc) == 0 {
		attestation.Mu.RUnlock()
		utils.SendError(encoder, "Attestation document not available")
		return
	}
	att := append([]byte(nil), attestation.Doc...)
	attestation.Mu.RUnlock()

	log.Printf("Attestation document size: %d bytes", len(att))

	res := enclaveproto.GetAttestationResponse{
		Attestation: base64.StdEncoding.EncodeToString(att),
	}
	utils.SendResponse(encoder, enclaveproto.Response[enclaveproto.GetAttestationResponse]{
		Success: true,
		Data:    res,
	})
}
