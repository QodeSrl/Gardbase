module github.com/QodeSrl/gardbase/pkg/crypto

go 1.24.4

require github.com/fxamacker/cbor/v2 v2.9.0

require (
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/sys v0.39.0 // indirect
)

require (
	github.com/QodeSrl/gardbase/pkg/enclaveproto v0.0.4
	golang.org/x/crypto v0.46.0
)

replace github.com/QodeSrl/gardbase/pkg/enclaveproto => ../enclaveproto
