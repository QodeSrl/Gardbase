module github.com/QodeSrl/gardbase/pkg/crypto

go 1.24.4

require github.com/fxamacker/cbor/v2 v2.5.0

require (
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/sys v0.38.0 // indirect
)

require (
	github.com/QodeSrl/gardbase/pkg/enclaveproto v0.0.0
	golang.org/x/crypto v0.45.0
)

replace github.com/QodeSrl/gardbase/pkg/enclaveproto => ../enclaveproto
