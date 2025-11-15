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
	EnclaveCID  uint32
	EnclavePort uint32
}

type EnclaveRequest struct {
	Type    string  `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
	ClientEphemeralPublicKey string `json:"client_ephemeral_public_key,omitempty"`
}

type EnclaveResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
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

func (p *VsockProxy) HandleDecrypt(c *gin.Context) {
	reqBody, err := c.GetRawData()
	if err != nil {
		c.JSON(400, gin.H{"error": "failed to read request body"})
		return
	}
	req := EnclaveRequest{
		Type: "decrypt",
		Payload: json.RawMessage(reqBody),
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
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	conn, err := vsock.Dial(p.EnclaveCID, p.EnclavePort, nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(timeout))

	_, err = conn.Write(append(payload, '\n'))
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		return scanner.Bytes(), nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("no response from enclave")
}
