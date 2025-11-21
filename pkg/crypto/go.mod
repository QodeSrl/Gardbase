module github.com/QodeSrl/gardbase/pkg/crypto

go 1.24.4

require github.com/aws/aws-sdk-go-v2/service/kms v1.48.2

require golang.org/x/sys v0.38.0 // indirect

require (
	github.com/QodeSrl/gardbase/pkg/enclaveproto v0.0.0
	github.com/aws/aws-sdk-go-v2 v1.39.6 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.13 // indirect
	github.com/aws/smithy-go v1.23.2 // indirect
	golang.org/x/crypto v0.45.0
)

replace github.com/QodeSrl/gardbase/pkg/enclaveproto => ../enclaveproto
