package handlers

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/gin-gonic/gin"
	"github.com/mdlayher/vsock"
)

type VsockProxy struct {
	// Enclave context ID
	EnclaveCID uint32
	// Enclave listening port
	EnclavePort uint32
	// KMS Client
	KMSClient *kms.Client
}

func (p *VsockProxy) HandleHealth(c *gin.Context) {
	req := enclaveproto.Request{
		Type:    "health",
		Payload: nil,
	}
	resBytes, err := p.sendToEnclave(req, 5*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var res enclaveproto.Response[json.RawMessage]
	if err := json.Unmarshal(resBytes, &res); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal health response: %v", err)})
		return
	}
	if !res.Success {
		c.JSON(500, gin.H{"error": res.Error})
		return
	}
	c.JSON(200, res)
}

func (p *VsockProxy) HandleSessionInit(c *gin.Context) {
	var initReq enclaveproto.SessionInitRequest
	if err := c.BindJSON(&initReq); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid session init request: %v", err)})
		return
	}
	payloadBytes, err := json.Marshal(initReq)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal session init request: %v", err)})
		return
	}
	req := enclaveproto.Request{
		Type:    "session_init",
		Payload: json.RawMessage(payloadBytes),
	}
	resBytes, err := p.sendToEnclave(req, 10*time.Second)
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
	c.JSON(200, res)
}

func (p *VsockProxy) HandleSessionUnwrap(c *gin.Context) {
	// Validate request (SessionUnwrapRequest)
	var unwrapReq enclaveproto.SessionUnwrapRequest
	if err := c.BindJSON(&unwrapReq); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid session unwrap request: %v", err)})
		return
	}

	// Build items for enclave request
	items := make([]enclaveproto.EnclaveSessionUnwrapItem, 0, len(unwrapReq.Items))

	// Request attestation document from enclave (GetAttestationRequest)
	attReq := enclaveproto.Request{
		Type: "get_attestation",
	}
	resAttBytes, err := p.sendToEnclave(attReq, 5*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var resAtt enclaveproto.Response[enclaveproto.GetAttestationResponse]
	if err := json.Unmarshal(resAttBytes, &resAtt); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal get attestation response: %v", err)})
		return
	}
	att, err := base64.StdEncoding.DecodeString(resAtt.Data.Attestation)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to decode attestation document: %v", err)})
		return
	}
	if !resAtt.Success {
		c.JSON(500, gin.H{"error": resAtt.Error})
		return
	}
	if len(att) == 0 {
		c.JSON(500, gin.H{"error": "Empty attestation document"})
		return
	}

	// Unwrap data keys with KMS using attestation document
	for _, it := range unwrapReq.Items {
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
		input := &kms.DecryptInput{
			CiphertextBlob: ctBytes,
			KeyId:          &unwrapReq.KeyId,
			Recipient: &kmsTypes.RecipientInfo{
				AttestationDocument:    att,
				KeyEncryptionAlgorithm: kmsTypes.KeyEncryptionMechanismRsaesOaepSha256,
			},
		}
		out, err := p.KMSClient.Decrypt(c.Request.Context(), input)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("KMS decrypt failed for object_id %s: %v", objId, err)})
			return
		}
		// append item for enclave unwrap request
		items = append(items, enclaveproto.EnclaveSessionUnwrapItem{
			ObjectId:   objId,
			Ciphertext: base64.StdEncoding.EncodeToString(out.CiphertextForRecipient),
		})
	}

	// Prepare enclave unwrap request
	unwrapEnclaveReq := enclaveproto.EnclaveSessionUnwrapRequest{
		SessionId: unwrapReq.SessionId,
		KeyId:     unwrapReq.KeyId,
		Items:     items,
	}
	payloadBytes, err := json.Marshal(unwrapEnclaveReq)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal session unwrap request: %v", err)})
		return
	}
	req := enclaveproto.Request{
		Type:    "session_unwrap",
		Payload: json.RawMessage(payloadBytes),
	}

	// Send unwrap request to enclave
	resBytes, err := p.sendToEnclave(req, 30*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var res enclaveproto.Response[enclaveproto.EnclaveSessionUnwrapResponse]
	if err := json.Unmarshal(resBytes, &res); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal session unwrap response: %v", err)})
		return
	}
	if !res.Success {
		c.JSON(500, gin.H{"error": res.Error})
		return
	}
	c.JSON(200, res)
}

func (p *VsockProxy) HandleSessionGenerateDEK(c *gin.Context) {
	// Validate request (SessionGenerateDEKRequest)
	var generateDEKReq enclaveproto.SessionGenerateDEKRequest
	if err := c.BindJSON(&generateDEKReq); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid session unwrap request: %v", err)})
		return
	}
	if generateDEKReq.Count <= 0 || generateDEKReq.Count > 100 {
		c.JSON(400, gin.H{"error": "Count must be between 1 and 100"})
		return
	}
	payloadBytes, err := json.Marshal(generateDEKReq)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal session unwrap request: %v", err)})
		return
	}

	// Request attestation document from enclave (GetAttestationRequest)
	attReq := enclaveproto.Request{
		Type: "get_attestation",
	}
	resAttBytes, err := p.sendToEnclave(attReq, 5*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var resAtt enclaveproto.Response[enclaveproto.GetAttestationResponse]
	if err := json.Unmarshal(resAttBytes, &resAtt); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal get attestation response: %v", err)})
		return
	}
	att, err := base64.StdEncoding.DecodeString(resAtt.Data.Attestation)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to decode attestation document: %v", err)})
		return
	}
	if !resAtt.Success {
		c.JSON(500, gin.H{"error": resAtt.Error})
		return
	}
	if len(att) == 0 {
		c.JSON(500, gin.H{"error": "Empty attestation document"})
		return
	}

	// Generate data keys with KMS using attestation document
	input := &kms.GenerateDataKeyInput{
		KeyId:   &generateDEKReq.KeyId,
		KeySpec: "AES_256",
		Recipient: &kmsTypes.RecipientInfo{
			AttestationDocument:    att,
			KeyEncryptionAlgorithm: kmsTypes.KeyEncryptionMechanismRsaesOaepSha256,
		},
	}
	deks := make([]enclaveproto.EnclaveDEKToPrepare, generateDEKReq.Count)
	for i := 0; i < generateDEKReq.Count; i++ {
		out, err := p.KMSClient.GenerateDataKey(c.Request.Context(), input)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to generate data key: %v", err)})
			return
		}
		if out.CiphertextForRecipient == nil {
			c.JSON(500, gin.H{"error": "No CiphertextForRecipient in KMS response"})
			return
		}
		deks[i] = enclaveproto.EnclaveDEKToPrepare{
			CiphertextBlob:         base64.StdEncoding.EncodeToString(out.CiphertextBlob),
			CiphertextForRecipient: base64.StdEncoding.EncodeToString(out.CiphertextForRecipient),
		}
		log.Printf("Generated DEK %d: CiphertextBlob len=%d, CiphertextForRecipient len=%d", i, len(out.CiphertextBlob), len(out.CiphertextForRecipient))
	}

	// Prepare DEK in enclave (EnclavePrepareDEKRequest)
	prepareDEKReq := enclaveproto.EnclavePrepareDEKRequest{
		DEKs:      deks,
		SessionId: generateDEKReq.SessionId,
	}
	payloadBytes, err = json.Marshal(prepareDEKReq)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal session unwrap request: %v", err)})
		return
	}
	req := enclaveproto.Request{
		Type:    "session_prepare_dek",
		Payload: json.RawMessage(payloadBytes),
	}
	resBytes, err := p.sendToEnclave(req, 30*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var prepRes enclaveproto.Response[enclaveproto.EnclavePrepareDEKResponse]
	if err := json.Unmarshal(resBytes, &prepRes); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal session unwrap response: %v", err)})
		return
	}
	if !prepRes.Success {
		c.JSON(500, gin.H{"error": prepRes.Error})
		return
	}

	// Build response
	res := enclaveproto.Response[enclaveproto.SessionGenerateDEKResponse]{
		Success: true,
		Data: enclaveproto.SessionGenerateDEKResponse{
			DEKs: prepRes.Data.DEKs,
		},
	}

	c.JSON(200, res)
}

func (p *VsockProxy) HandleDecrypt(c *gin.Context) {
	var decryptReq enclaveproto.DecryptRequest
	if err := c.BindJSON(&decryptReq); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid decrypt request: %v", err)})
		return
	}
	payloadBytes, err := json.Marshal(decryptReq)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal decrypt request: %v", err)})
		return
	}
	// build enclave request
	req := enclaveproto.Request{
		Type:    "decrypt",
		Payload: json.RawMessage(payloadBytes),
	}
	resBytes, err := p.sendToEnclave(req, 15*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var res enclaveproto.Response[enclaveproto.DecryptResponse]
	if err := json.Unmarshal(resBytes, &res); err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to unmarshal decrypt response: %v", err)})
		return
	}
	if !res.Success {
		c.JSON(500, gin.H{"error": res.Error})
		return
	}
	c.JSON(200, res)
}

func (p *VsockProxy) sendToEnclave(req enclaveproto.Request, timeout time.Duration) (json.RawMessage, error) {
	jsonReq, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal request: %v", err)
	}

	// connect to enclave via vsock
	conn, err := vsock.Dial(p.EnclaveCID, p.EnclavePort, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to enclave vsock: %v", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(timeout))

	// send request
	if _, err = conn.Write(append(jsonReq, '\n')); err != nil {
		return nil, fmt.Errorf("Failed to send request to enclave: %v", err)
	}

	// read response through scanner
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("Failed to read response from enclave: %v", err)
		}
		return nil, fmt.Errorf("no response from enclave")
	}

	return scanner.Bytes(), nil
}
