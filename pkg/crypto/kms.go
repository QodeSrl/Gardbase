package crypto

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/kms"
)

func GenerateDEK(ctx context.Context, kmsClient *kms.Client, keyID string) (DEK []byte, encryptedDEK []byte, err error) {
	generateDEKInput := &kms.GenerateDataKeyInput{
		KeyId:   &keyID,
		KeySpec: "AES_256",
	}
	generateDEKOutput, err := kmsClient.GenerateDataKey(ctx, generateDEKInput)
	if err != nil {
		return nil, nil, err
	}
	return generateDEKOutput.CiphertextForRecipient, generateDEKOutput.CiphertextBlob, nil
}
