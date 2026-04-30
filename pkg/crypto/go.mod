module github.com/qodesrl/gardbase/pkg/crypto

go 1.24.4

require github.com/fxamacker/cbor/v2 v2.9.0

require (
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/sys v0.40.0 // indirect
)

require (
	github.com/qodesrl/gardbase/pkg/api v0.1.1
	github.com/qodesrl/gardbase/pkg/enclaveproto v0.1.1
	golang.org/x/crypto v0.47.0
)

replace github.com/qodesrl/gardbase/pkg/enclaveproto => ../enclaveproto

replace github.com/qodesrl/gardbase/pkg/api => ../api
