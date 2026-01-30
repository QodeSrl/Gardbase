module github.com/QodeSrl/gardbase/apps/enclave-service

go 1.24.4

require (
	github.com/QodeSrl/gardbase/pkg/enclaveproto v0.0.4
	github.com/hf/nsm v0.0.0-20220930140112-cd181bd646b9
	github.com/mdlayher/vsock v1.2.1
	golang.org/x/crypto v0.46.0
)

require (
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/mdlayher/socket v0.5.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
)

replace github.com/QodeSrl/gardbase/pkg/enclaveproto => ../../pkg/enclaveproto
