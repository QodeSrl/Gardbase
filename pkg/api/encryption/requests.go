package encryption

import "github.com/QodeSrl/gardbase/pkg/enclaveproto"

type SessionInitRequest = enclaveproto.SessionInitRequest

type SessionUnwrapRequest = enclaveproto.SessionUnwrapRequest

type SessionGenerateDEKRequest struct {
	// Session ID, Base64-encoded
	SessionId string `json:"session_id"`
	// Number of DEKs to generate
	Count int `json:"count"`
}

type DecryptRequest = enclaveproto.DecryptRequest
