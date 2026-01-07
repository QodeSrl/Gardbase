package services

import (
	"github.com/QodeSrl/gardbase/apps/api/internal/storage"
	"github.com/aws/aws-sdk-go-v2/service/kms"
)

type KeyManagementService struct {
	kmsClient   *kms.Client
	kmsKeyID    string
	dynamo      *storage.DynamoClient
	EnclaveCID  uint32
	EnclavePort uint32
}

func NewKeyManagementService(kmsClient *kms.Client, dynamo *storage.DynamoClient, enclaveCID uint32, enclavePort uint32) *KeyManagementService {
	return &KeyManagementService{
		kmsClient:   kmsClient,
		dynamo:      dynamo,
		EnclaveCID:  enclaveCID,
		EnclavePort: enclavePort,
	}
}

type TenantKeys struct {
	TenantID  string
	MasterKey string
	TableSalt string
	Version   int
}
