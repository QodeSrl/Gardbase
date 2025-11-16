package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mdlayher/vsock"
)

type VsockProxy struct {
	// Enclave context ID
	EnclaveCID  uint32
	// Enclave listening port
	EnclavePort uint32
}

type EnclaveRequest struct {
	// "health", "attestation", "decrypt", etc.
	Type                     string          `json:"type,omitempty"`
	// Request-specific payload 
	Payload                  json.RawMessage `json:"payload"`
	// Client's ephemeral public key 
	ClientEphemeralPublicKey string          `json:"client_ephemeral_public_key"`
}

type EnclaveResponse struct {
	Success bool   `json:"success,omitempty"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error"`
}

func (p *VsockProxy) HandleHealth(c *gin.Context) {
	req := EnclaveRequest{
		Type: "health",
		Payload: nil,
		ClientEphemeralPublicKey: c.GetHeader("X-Client-Ephemeral-Public-Key"),
	}
	res, err := p.sendToEnclave(req, 5*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Data(200, "application/json", res)
}

func (p *VsockProxy) HandleAttestation(c *gin.Context) {
	req := EnclaveRequest{
		Type: "attestation",
		Payload: nil,
		ClientEphemeralPublicKey: c.GetHeader("X-Client-Ephemeral-Public-Key"),
	}
	res, err := p.sendToEnclave(req, 10*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Data(200, "application/json", res)
}

type DecryptRequest struct {
	// Encrypted DEK, Base64-encoded
	Ciphertext string `json:"ciphertext,omitempty"`
	// Request nonce, Base64-encoded
	Nonce      string `json:"nonce,omitempty"`
	// KMS Key ID
	KeyID 	   string `json:"key_id,omitempty"`
}

func (p *VsockProxy) HandleDecrypt(c *gin.Context) {
	var decryptReq  DecryptRequest
	if err := c.BindJSON(&decryptReq); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid decrypt request: %v", err)})
		return
	}
	if decryptReq.Ciphertext == "" {
		c.JSON(400, gin.H{"error": "Ciphertext is required"})
		return
	}
	payloadBytes, err := json.Marshal(decryptReq)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal decrypt request: %v", err)})
		return
	}
	// build enclave request
	req := EnclaveRequest{
		Type: "decrypt",
		Payload: json.RawMessage(payloadBytes),
		ClientEphemeralPublicKey: c.GetHeader("X-Client-Ephemeral-Public-Key"),
	}
	res, err := p.sendToEnclave(req, 15*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Data(200, "application/json", res)
}

func (p *VsockProxy) sendToEnclave(req EnclaveRequest, timeout time.Duration) ([]byte, error) {
	jsonReq, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// connect to enclave via vsock
	conn, err := vsock.Dial(p.EnclaveCID, p.EnclavePort, nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(timeout))

	// send request
	_, err = conn.Write(append(jsonReq, '\n'))
	if err != nil {
		return nil, err
	}

	// read response through scanner 
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		return scanner.Bytes(), nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("no response from enclave")
}
