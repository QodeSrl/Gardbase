package encryption

import "github.com/QodeSrl/gardbase/pkg/enclaveproto"

type SessionInitResponse = enclaveproto.SessionInitResponse

type SessionUnwrapResponse = enclaveproto.SessionUnwrapResponse

type SessionGenerateDEKResponse struct {
	// List of generated DEKs
	DEKs []enclaveproto.GeneratedDEK `json:"deks"`
	// Index encryption key (IEK), Base64-encoded
	SealedIEK string `json:"iek"`
	// IEK Nonce used for sealing, Base64-encoded
	IEKNonce string `json:"iek_nonce"`
}

type DecryptResponse = enclaveproto.DecryptResponse
