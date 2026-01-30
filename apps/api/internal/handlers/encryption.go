package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/QodeSrl/gardbase/apps/api/internal/services"
	"github.com/QodeSrl/gardbase/apps/api/internal/storage"
	"github.com/QodeSrl/gardbase/pkg/api/encryption"
	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/gin-gonic/gin"
)

type EncryptionHandler struct {
	Vsock  *services.Vsock
	KMS    *services.KMS
	Dynamo *storage.DynamoClient
}

func (e *EncryptionHandler) HandleSessionInit(c *gin.Context) {
	var req encryption.SessionInitRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid session init request: %v", err)})
		return
	}
	// SessionInitRequest = enclaveproto.SessionInitRequest
	payloadBytes, err := json.Marshal(req)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal session init request: %v", err)})
		return
	}
	enclaveReq := enclaveproto.Request{
		Type:    "session_init",
		Payload: json.RawMessage(payloadBytes),
	}
	resBytes, err := e.Vsock.SendToEnclave(enclaveReq, 10*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var res enclaveproto.Response[enclaveproto.SessionInitResponse]
	if err := json.Unmarshal(resBytes, &res); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal session init response: %v", err)})
		return
	}
	if !res.Success {
		c.JSON(500, gin.H{"error": res.Error})
		return
	}
	c.JSON(200, res.Data)
}

func (e *EncryptionHandler) HandleSessionUnwrap(c *gin.Context) {
	// Validate request (SessionUnwrapRequest)
	var req encryption.SessionUnwrapRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid session unwrap request: %v", err)})
		return
	}

	// Build items for enclave request
	items := make([]enclaveproto.SessionUnwrapItem, 0, len(req.Items))

	// Request attestation document from enclave (GetAttestationRequest)
	att, err := e.Vsock.RequestAttestationDocument()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
	}

	// Unwrap data keys with KMS using attestation document
	for _, it := range req.Items {
		objId := it.ObjectId
		if objId == "" {
			c.JSON(400, gin.H{"error": "missing object_id for one of the items"})
			return
		}
		if it.Ciphertext == "" {
			c.JSON(400, gin.H{"error": fmt.Sprintf("missing ciphertext for object_id %s", objId)})
			return
		}
		ctBytes, err := base64.StdEncoding.DecodeString(it.Ciphertext)
		if err != nil {
			c.JSON(400, gin.H{"error": fmt.Sprintf("invalid base64 ciphertext for object_id %s: %v", objId, err)})
			return
		}
		tenantId := c.GetString("tenantId")
		out, err := e.KMS.Decrypt(c.Request.Context(), ctBytes, att, tenantId, services.PurposeDataKey)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("KMS decrypt failed for object_id %s: %v", objId, err)})
			return
		}
		// append item for enclave unwrap request
		items = append(items, enclaveproto.SessionUnwrapItem{
			ObjectId:   objId,
			Ciphertext: base64.StdEncoding.EncodeToString(out.CiphertextForRecipient),
		})
	}

	// Prepare enclave unwrap request
	enclaveReqBody := enclaveproto.SessionUnwrapRequest{
		SessionId: req.SessionId,
		Items:     items,
	}
	payloadBytes, err := json.Marshal(enclaveReqBody)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal session unwrap request: %v", err)})
		return
	}
	enclaveReq := enclaveproto.Request{
		Type:    "session_unwrap",
		Payload: json.RawMessage(payloadBytes),
	}

	// Send unwrap request to enclave
	resBytes, err := e.Vsock.SendToEnclave(enclaveReq, 30*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var enclaveRes enclaveproto.Response[enclaveproto.SessionUnwrapResponse]
	if err := json.Unmarshal(resBytes, &enclaveRes); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal session unwrap response: %v", err)})
		return
	}
	if !enclaveRes.Success {
		c.JSON(500, gin.H{"error": enclaveRes.Error})
		return
	}

	c.JSON(200, enclaveRes.Data)
}

func (e *EncryptionHandler) HandleSessionGenerateDEK(c *gin.Context) {
	// Validate request (SessionGenerateDEKRequest)
	var req encryption.SessionGenerateDEKRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid session unwrap request: %v", err)})
		return
	}
	if req.Count <= 0 || req.Count > 100 {
		c.JSON(400, gin.H{"error": "Count must be between 1 and 100"})
		return
	}
	payloadBytes, err := json.Marshal(req)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal session unwrap request: %v", err)})
		return
	}

	// Request attestation document from enclave (GetAttestationRequest)
	att, err := e.Vsock.RequestAttestationDocument()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	tenantId := c.GetString("tenantId")
	tenant, err := e.Dynamo.GetTenant(c.Request.Context(), tenantId)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get tenant config"})
		return
	}

	// Generate data keys with KMS using attestation document
	deks := make([]enclaveproto.DEKToPrepare, req.Count)
	for i := 0; i < req.Count; i++ {
		out, err := e.KMS.GenerateDataKey(c.Request.Context(), att, tenantId, services.PurposeDataKey)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to generate data key: %v", err)})
			return
		}
		if out.CiphertextForRecipient == nil {
			c.JSON(500, gin.H{"error": "No CiphertextForRecipient in KMS response"})
			return
		}
		deks[i] = enclaveproto.DEKToPrepare{
			CiphertextBlob:         base64.StdEncoding.EncodeToString(out.CiphertextBlob),
			CiphertextForRecipient: base64.StdEncoding.EncodeToString(out.CiphertextForRecipient),
		}
		log.Printf("Generated DEK %d: CiphertextBlob len=%d, CiphertextForRecipient len=%d", i, len(out.CiphertextBlob), len(out.CiphertextForRecipient))
	}

	KMSWrappedMasterKey, err := base64.StdEncoding.DecodeString(tenant.WrappedMasterKey)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to decode wrapped master key: %v", err)})
		return
	}

	wrappedMasterkeyOut, err := e.KMS.Decrypt(c.Request.Context(), KMSWrappedMasterKey, att, tenantId, services.PurposeMasterKey)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to decrypt wrapped master key: %v", err)})
		return
	}

	// Prepare DEK in enclave (PrepareDEKRequest)
	prepareDEKReq := enclaveproto.PrepareDEKRequest{
		DEKs:             deks,
		WrappedMasterKey: base64.StdEncoding.EncodeToString(wrappedMasterkeyOut.CiphertextForRecipient),
		SessionId:        req.SessionId,
	}
	payloadBytes, err = json.Marshal(prepareDEKReq)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal session unwrap request: %v", err)})
		return
	}
	enclaveReq := enclaveproto.Request{
		Type:    "session_prepare_dek",
		Payload: json.RawMessage(payloadBytes),
	}
	resBytes, err := e.Vsock.SendToEnclave(enclaveReq, 30*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var prepRes enclaveproto.Response[enclaveproto.PrepareDEKResponse]
	if err := json.Unmarshal(resBytes, &prepRes); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal session unwrap response: %v", err)})
		return
	}
	if !prepRes.Success {
		c.JSON(500, gin.H{"error": prepRes.Error})
		return
	}

	c.JSON(200, prepRes.Data)
}

func (e *EncryptionHandler) HandleDecrypt(c *gin.Context) {
	// Validate request (DecryptRequest)
	var decryptReq encryption.DecryptRequest
	if err := c.BindJSON(&decryptReq); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid decrypt request: %v", err)})
		return
	}
	tenantId := c.GetString("tenantId")

	// Request attestation document from enclave (GetAttestationRequest)
	att, err := e.Vsock.RequestAttestationDocument()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	ciphertext, err := base64.StdEncoding.DecodeString(decryptReq.Ciphertext)
	if err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid ciphertext encoding: %v", err)})
		return
	}

	// Unwrap data key with KMS using attestation document
	out, err := e.KMS.Decrypt(c.Request.Context(), ciphertext, att, tenantId, services.PurposeDataKey)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("KMS decrypt failed: %v", err)})
		return
	}

	// Prepare decrypt request for enclave
	decryptReq.Ciphertext = base64.StdEncoding.EncodeToString(out.CiphertextForRecipient)
	payloadBytes, err := json.Marshal(decryptReq)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal decrypt request: %v", err)})
		return
	}
	req := enclaveproto.Request{
		Type:    "decrypt",
		Payload: json.RawMessage(payloadBytes),
	}

	// Send decrypt request to enclave
	resBytes, err := e.Vsock.SendToEnclave(req, 15*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var enclaveRes enclaveproto.Response[enclaveproto.DecryptResponse]
	if err := json.Unmarshal(resBytes, &enclaveRes); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal decrypt response: %v", err)})
		return
	}
	if !enclaveRes.Success {
		c.JSON(500, gin.H{"error": enclaveRes.Error})
		return
	}
	c.JSON(200, enclaveRes.Data)
}
