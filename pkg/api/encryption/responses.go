package encryption

import "github.com/QodeSrl/gardbase/pkg/enclaveproto"

type SessionInitResponse = enclaveproto.SessionInitResponse

type SessionUnwrapResponse = enclaveproto.SessionUnwrapResponse

type SessionGenerateDEKResponse = enclaveproto.PrepareDEKResponse

type DecryptResponse = enclaveproto.DecryptResponse
