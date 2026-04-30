package enclaveproto

type PrepareIEKRequest struct {
	// Session ID
	SessionId string `json:"session_id"`
	// IEK to prepare
	IEK []byte `json:"iek"`
}

type PrepareIEKResponse struct {
	// Index encryption key (IEK)
	SealedIEK []byte `json:"iek"`
	// IEK Nonce used for sealing
	IEKNonce []byte `json:"iek_nonce"`
}
