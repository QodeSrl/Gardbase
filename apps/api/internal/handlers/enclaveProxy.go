package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/gin-gonic/gin"
	"github.com/mdlayher/vsock"
)

type VsockProxy struct {
	// Enclave context ID
	EnclaveCID uint32
	// Enclave listening port
	EnclavePort uint32
}

func (p *VsockProxy) HandleHealth(c *gin.Context) {
	req := enclaveproto.Request{
		Type:    "health",
		Payload: nil,
	}
	res, err := p.sendToEnclave(req, 5*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Data(200, "application/json", res)
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
	res, err := p.sendToEnclave(req, 10*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Data(200, "application/json", res)
}

func (p *VsockProxy) HandleSessionUnwrap(c *gin.Context) {
	var unwrapReq enclaveproto.SessionUnwrapRequest
	if err := c.BindJSON(&unwrapReq); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Invalid session unwrap request: %v", err)})
		return
	}
	payloadBytes, err := json.Marshal(unwrapReq)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to marshal session unwrap request: %v", err)})
		return
	}
	req := enclaveproto.Request{
		Type:    "session_unwrap",
		Payload: json.RawMessage(payloadBytes),
	}
	res, err := p.sendToEnclave(req, 30*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Data(200, "application/json", res)
}

func (p *VsockProxy) HandleSessionGenerateDEK(c *gin.Context) {
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
	req := enclaveproto.Request{
		Type:    "session_generate_dek",
		Payload: json.RawMessage(payloadBytes),
	}
	res, err := p.sendToEnclave(req, 30*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Data(200, "application/json", res)
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
	res, err := p.sendToEnclave(req, 15*time.Second)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.Data(200, "application/json", res)
}

func (p *VsockProxy) sendToEnclave(req enclaveproto.Request, timeout time.Duration) ([]byte, error) {
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
