package encryption

import "github.com/QodeSrl/gardbase/pkg/enclaveproto"

type SessionInitRequest = enclaveproto.SessionInitRequest

type SessionUnwrapRequest = enclaveproto.SessionUnwrapRequest

type SessionGenerateDEKRequest struct {
	// Session ID
	SessionId string `json:"session_id"`
	// Number of DEKs to generate
	Count int `json:"count"`
	// Hash of the table for which DEKs are generated
	TableHash string `json:"table_hash"`
}

type SessionGetTableIEKRequest struct {
	// Session ID
	SessionId string `json:"session_id"`
	// Hash of the table for which IEK is requested
	TableHash string `json:"table_hash"`
}

type SessionGetTableIEKResponse = enclaveproto.PrepareIEKResponse

type DecryptRequest = enclaveproto.DecryptRequest
