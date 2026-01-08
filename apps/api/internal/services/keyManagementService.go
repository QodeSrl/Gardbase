package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/QodeSrl/gardbase/apps/api/internal/storage"
	"github.com/QodeSrl/gardbase/pkg/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/google/uuid"
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

func (k *KeyManagementService) GenerateTenantKeys(ctx context.Context, tenantID string) (*TenantKeys, error) {
	masterKey := make([]byte, 32)
	if _, err := rand.Read(masterKey); err != nil {
		return nil, err
	}

	tableSalt := make([]byte, 32)
	if _, err := rand.Read(tableSalt); err != nil {
		return nil, err
	}

	wrappedMasterKey, err := k.wrapKey(ctx, masterKey, tenantID, "master-key")
	if err != nil {
		return nil, err
	}

	wrappedTableSalt, err := k.wrapKey(ctx, tableSalt, tenantID, "table-salt")
	if err != nil {
		return nil, err
	}

	config := models.NewTenantConfig(tenantID, wrappedMasterKey, wrappedTableSalt, 1, uuid.New().String())

	if err := k.dynamo.StoreTenantConfig(ctx, config); err != nil {
		return nil, err
	}

	// Warning: this is the only time we return plaintext keys (only time they are exposed)
	return &TenantKeys{
		TenantID:  tenantID,
		MasterKey: base64.StdEncoding.EncodeToString(masterKey),
		TableSalt: base64.StdEncoding.EncodeToString(tableSalt),
		Version:   1,
	}, nil
}

// wrapKey uses AWS KMS to encrypt the given plaintext key with the specified tenant ID and purpose.
func (k *KeyManagementService) wrapKey(ctx context.Context, pt []byte, tenantID string, purpose string) ([]byte, error) {
	input := &kms.EncryptInput{
		KeyId:     aws.String(k.kmsKeyID),
		Plaintext: pt,
		EncryptionContext: map[string]string{
			"tenant_id": tenantID,
			"purpose":   purpose,
		},
	}

	result, err := k.kmsClient.Encrypt(ctx, input)
	if err != nil {
		return nil, err
	}

	return result.CiphertextBlob, nil
}
