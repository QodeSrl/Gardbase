package enclaveproto

type GetAttestationRequest struct {
}

type GetAttestationResponse struct {
	// Attestation document
	Attestation []byte `json:"attestation"`
}
