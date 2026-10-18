package handlers

import (
	"fmt"
	"log"

	"github.com/qodesrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/qodesrl/gardbase/pkg/enclaveproto"
)

func HandleGetAttestation(attestation *utils.Attestation) (*enclaveproto.GetAttestationResponse, error) {
	attestation.Mu.RLock()
	if len(attestation.Doc) == 0 {
		attestation.Mu.RUnlock()
		return nil, fmt.Errorf("Attestation document not available")
	}
	att := append([]byte(nil), attestation.Doc...)
	attestation.Mu.RUnlock()

	log.Printf("Attestation document size: %d bytes", len(att))

	res := enclaveproto.GetAttestationResponse{
		Attestation: att,
	}
	return &res, nil
}
