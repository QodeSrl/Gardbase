package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/session"
	"github.com/QodeSrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"golang.org/x/crypto/chacha20poly1305"
)

func HandleSessionGenerateTableHash(encoder *json.Encoder, payload json.RawMessage) {
	var req enclaveproto.SessionGenerateTableHashRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		utils.SendError(encoder, fmt.Sprintf("Invalid generate table hash request: %v", err))
		return
	}

	sess, ok := session.GetSession(req.SessionID)
	if !ok {
		utils.SendError(encoder, "Invalid or expired session ID")
		return
	}

	sessAead, err := chacha20poly1305.NewX(sess.Key)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to create AEAD cipher: %v", err))
		return
	}

	tableName, err := sessAead.Open(nil, req.SessionTableNameNonce, req.SessionEncryptedTableName, nil)
	if err != nil {
		utils.SendError(encoder, fmt.Sprintf("Failed to decrypt table name: %v", err))
		return
	}

	tableHash := utils.Hash(tableName, req.TableSalt)
	res := enclaveproto.SessionGenerateTableHashResponse{
		TableHash: tableHash,
	}
	utils.SendResponse(encoder, enclaveproto.Response[enclaveproto.SessionGenerateTableHashResponse]{
		Success: true,
		Data:    res,
	})
}
