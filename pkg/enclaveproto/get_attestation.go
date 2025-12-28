package enclaveproto

type GetAttestationRequest struct {
}

type GetAttestationResponse struct {
	// Attestation document, Base64-encoded
	Attestation string `json:"attestation"`
}
