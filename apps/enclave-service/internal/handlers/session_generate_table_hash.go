package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/qodesrl/gardbase/apps/enclave-service/internal/session"
	"github.com/qodesrl/gardbase/apps/enclave-service/internal/utils"
	"github.com/qodesrl/gardbase/pkg/enclaveproto"
	"golang.org/x/crypto/chacha20poly1305"
)

func HandleSessionGenerateTableHash(payload json.RawMessage) (res enclaveproto.SessionGenerateTableHashResponse, err error) {
	var req enclaveproto.SessionGenerateTableHashRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return res, fmt.Errorf("Invalid generate table hash request: %v", err)
	}

	sess, ok := session.GetSession(req.SessionID)
	if !ok {
		return res, fmt.Errorf("Invalid or expired session ID")
	}

	sessAead, err := chacha20poly1305.NewX(sess.Key)
	if err != nil {
		return res, fmt.Errorf("Failed to create AEAD cipher: %v", err)
	}

	tableName, err := sessAead.Open(nil, req.SessionTableNameNonce, req.SessionEncryptedTableName, nil)
	if err != nil {
		return res, fmt.Errorf("Failed to decrypt table name: %v", err)
	}

	tableHash := utils.Hash(tableName, req.TableSalt)
	res = enclaveproto.SessionGenerateTableHashResponse{
		TableHash: tableHash,
	}
	return res, nil
}
