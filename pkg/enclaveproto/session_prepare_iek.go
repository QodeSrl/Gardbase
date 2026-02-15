package enclaveproto

type PrepareIEKRequest struct {
	// Session ID, Base64-encoded
	SessionId string `json:"session_id"`
	// IEK to prepare, Base64-encoded
	IEK string `json:"iek"`
}

type PrepareIEKResponse struct {
	// Index encryption key (IEK), Base64-encoded
	SealedIEK string `json:"iek"`
	// IEK Nonce used for sealing, Base64-encoded
	IEKNonce string `json:"iek_nonce"`
}
