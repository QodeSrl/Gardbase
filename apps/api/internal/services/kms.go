package services

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
)

type KMS struct {
	Client *kms.Client
	KeyID  string
}

func NewKMSService(ctx context.Context, cfg aws.Config, keyID string, useLocalstack bool, localstackUrl string) *KMS {
	var kmsClient *kms.Client
	if useLocalstack {
		kmsClient = kms.NewFromConfig(cfg, func(o *kms.Options) {
			if useLocalstack {
				o.BaseEndpoint = aws.String(localstackUrl)
			}
		})
	} else {
		kmsClient = kms.NewFromConfig(cfg)
	}
	return &KMS{
		Client: kmsClient,
		KeyID:  keyID,
	}
}

func (k *KMS) GenerateDataKey(ctx context.Context, attestationDocument []byte, tenantID string, purpose string) (*kms.GenerateDataKeyOutput, error) {
	return k.Client.GenerateDataKey(ctx, &kms.GenerateDataKeyInput{
		KeyId:   aws.String(k.KeyID),
		KeySpec: kmsTypes.DataKeySpecAes256,
		Recipient: &kmsTypes.RecipientInfo{
			AttestationDocument:    attestationDocument,
			KeyEncryptionAlgorithm: kmsTypes.KeyEncryptionMechanismRsaesOaepSha256,
		},
		EncryptionContext: map[string]string{
			"tenant_id": tenantID,
			"purpose":   purpose,
		},
	})
}

func (k *KMS) Decrypt(ctx context.Context, ciphertextBlob []byte, attestationDocument []byte, tenantID string, purpose string) (*kms.DecryptOutput, error) {
	return k.Client.Decrypt(ctx, &kms.DecryptInput{
		KeyId:          aws.String(k.KeyID),
		CiphertextBlob: ciphertextBlob,
		Recipient: &kmsTypes.RecipientInfo{
			AttestationDocument:    attestationDocument,
			KeyEncryptionAlgorithm: kmsTypes.KeyEncryptionMechanismRsaesOaepSha256,
		},
		EncryptionContext: map[string]string{
			"tenant_id": tenantID,
			"purpose":   purpose,
		},
	})
}
