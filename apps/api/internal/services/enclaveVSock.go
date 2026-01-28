package services

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/QodeSrl/gardbase/pkg/enclaveproto"
	"github.com/gin-gonic/gin"
	"github.com/mdlayher/vsock"
)

type Vsock struct {
	// Enclave context ID
	EnclaveCID uint32
	// Enclave listening port
	EnclavePort uint32
}

func (v *Vsock) RequestAttestationDocument() ([]byte, error) {
	req := enclaveproto.Request{
		Type: "get_attestation",
	}
	resBytes, err := v.SendToEnclave(req, 5*time.Second)
	if err != nil {
		return nil, err
	}
	var res enclaveproto.Response[enclaveproto.GetAttestationResponse]
	if err := json.Unmarshal(resBytes, &res); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal get attestation response: %v", err)
	}
	att, err := base64.StdEncoding.DecodeString(res.Data.Attestation)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode attestation document: %v", err)
	}
	if !res.Success {
		return nil, fmt.Errorf("Enclave returned error: %s", res.Error)
	}
	if len(att) == 0 {
		return nil, fmt.Errorf("Empty attestation document")
	}
	return att, nil
}

func (v *Vsock) SendToEnclave(req enclaveproto.Request, timeout time.Duration) (json.RawMessage, error) {
	jsonReq, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal request: %v", err)
	}

	// connect to enclave via vsock
	conn, err := vsock.Dial(v.EnclaveCID, v.EnclavePort, nil)
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

func (v *Vsock) HandleHealth(c *gin.Context) {
	req := enclaveproto.Request{
		Type:    "health",
		Payload: nil,
	}
	resBytes, err := v.SendToEnclave(req, 5*time.Second)
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
