package handlers

import (
	"encoding/base64"
	"fmt"

	"github.com/QodeSrl/gardbase/apps/api/internal/services"
	"github.com/QodeSrl/gardbase/apps/api/internal/storage"
	"github.com/QodeSrl/gardbase/pkg/api/tenants"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TenantHandler struct {
	Vsock  *services.Vsock
	KMS    *services.KMS
	Dynamo *storage.DynamoClient
}

func (t *TenantHandler) HandleCreateTenant(c *gin.Context) {
	tenantID := uuid.NewString()
	var req tenants.CreateTenantRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid create tenant request: %v", err)})
		return
	}

	att, err := t.Vsock.RequestAttestationDocument()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// TODO: implement master key recovery mechanism
	masterKeyRes, err := t.KMS.GenerateDataKey(c.Request.Context(), att, tenantID, services.PurposeMasterKey)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to generate master key: %v", err)})
		return
	}

	tableSaltRes, err := t.KMS.GenerateDataKey(c.Request.Context(), att, tenantID, services.PurposeTableSalt)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to generate table salt: %v", err)})
		return
	}

	// prepareMasterKeyReq := enclaveproto.EnclavePrepareKEKRequest{
	// 	ClientEphemeralPublicKey: req.ClientPubKey,
	// 	MasterKey:                base64.StdEncoding.EncodeToString(masterKeyRes.CiphertextForRecipient),
	// 	TableSalt:                base64.StdEncoding.EncodeToString(tableSaltRes.CiphertextForRecipient),
	// }

	// payloadBytes, err := json.Marshal(prepareMasterKeyReq)
	// if err != nil {
	// 	c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal prepare KEK request: %v", err)})
	// 	return
	// }

	// reqEnclave := enclaveproto.Request{
	// 	Type:    "prepare_kek",
	// 	Payload: json.RawMessage(payloadBytes),
	// }
	// resBytes, err := p.sendToEnclave(reqEnclave, 15*time.Second)
	// if err != nil {
	// 	c.JSON(500, gin.H{"error": err.Error()})
	// 	return
	// }
	// var res enclaveproto.Response[enclaveproto.EnclavePrepareKEKResponse]
	// if err := json.Unmarshal(resBytes, &res); err != nil {
	// 	c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal prepare KEK response: %v", err)})
	// 	return
	// }
	// if !res.Success {
	// 	c.JSON(500, gin.H{"error": res.Error})
	// 	return
	// }

	err = t.Dynamo.CreateTenant(c.Request.Context(), tenantID, masterKeyRes.CiphertextBlob, tableSaltRes.CiphertextBlob)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to create tenant: %v", err)})
		return
	}
	// TODO: Implement key recovery mechanism
	apiKey, err := t.Dynamo.CreateAPIKey(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to create API key: %v", err)})
		return
	}

	c.JSON(200, tenants.CreateTenantResponse{
		TenantID: tenantID,
		// TODO: later on, implement advanced self-managed keys
		// EncryptedMasterKey:  res.Data.MasterKey,
		// EncryptedTableSalt:  res.Data.TableSalt,
		// EnclavePubKey:       res.Data.EnclavePubKey,
		AttestationDocument: base64.StdEncoding.EncodeToString(att),
		// Nonce:               res.Data.Nonce,
		APIKey: apiKey,
	})
}
