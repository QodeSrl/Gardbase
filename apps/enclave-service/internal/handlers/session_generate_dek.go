package handlers

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/session"
	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/hf/nsm"
	"golang.org/x/crypto/chacha20poly1305"
)

func HandleSessionGenerateDEK(encoder *json.Encoder, payload json.RawMessage, nsmSession *nsm.Session, nsmPrivKey *rsa.PrivateKey, nsmAttestation []byte, kmsClient *kms.Client) {
	var req enclaveproto.SessionGenerateDEKRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid session generate DEK request: %v", err))
		return
	}

	sess, ok := session.GetSession(req.SessionId)
	if !ok {
		utils.SendError(encoder, "Invalid or expired session ID")
		return
	}

	aead, err := chacha20poly1305.NewX(sess.Key)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to create AEAD cipher: %v", err))
		return
	}

	results := make([]enclaveproto.GeneratedDEK, 0, req.Count)
	ctx := context.Background()

	input := &kms.GenerateDataKeyInput{
		KeyId:   &req.KeyId,
		KeySpec: "AES_256",
		Recipient: &kmsTypes.RecipientInfo{
			AttestationDocument:    nsmAttestation,
			KeyEncryptionAlgorithm: "RSAES_OAEP_SHA_256",
		},
	}

	for i := 0; i < req.Count; i++ {
		output, err := kmsClient.GenerateDataKey(ctx, input)
		if err != nil {
			utils.SendError(encoder, fmt.Sprintf("Failed to generate data key: %v", err))
			return
		}
		// note: here nsmSession is used as a rand.Reader
		plainDEK, err := rsa.DecryptOAEP(sha256.New(), nsmSession, nsmPrivKey, output.CiphertextForRecipient, nil)
		if err != nil {
			utils.SendError(encoder, fmt.Sprintf("Failed to decrypt data key: %v", err))
			return
		}

		// nonce for session encryption
		nonce := make([]byte, chacha20poly1305.NonceSizeX) // 24 bytes
		if _, err := nsmSession.Read(nonce); err != nil {
			utils.SendError(encoder, fmt.Sprintf("Failed to read nonce: %v", err))
			return
		}

		// seal DEK with session key
		sealedDEK := aead.Seal(nil, nonce, plainDEK, nil)

		utils.Zero(plainDEK)

		results = append(results, enclaveproto.GeneratedDEK{
			SealedDEK:       base64.StdEncoding.EncodeToString(sealedDEK),
			KmsEncryptedDEK: base64.StdEncoding.EncodeToString(output.CiphertextBlob),
			Nonce:           base64.StdEncoding.EncodeToString(nonce),
		})
	}

	res := enclaveproto.SessionGenerateDEKResponse{
		DEKs: results,
	}

	utils.SendResponse(encoder, enclaveproto.Response[enclaveproto.SessionGenerateDEKResponse]{
		Success: true,
		Data:    res,
	})
}
