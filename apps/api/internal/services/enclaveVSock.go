package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/qodesrl/gardbase/pkg/enclaveproto"
)

type VsockPool struct {
	cid     uint32
	port    uint32
	mu      sync.Mutex
	idle    []*vsockConn
	maxIdle int
}

type vsockConn struct {
	conn    net.Conn
	scanner *bufio.Scanner
	encoder *json.Encoder
}

type Vsock struct {
	// Enclave context ID
	EnclaveCID uint32
	// Enclave listening port
	EnclavePort uint32
	// Pool of vsock connections for reuse
	pool VsockPool
}

func NewVsock(enclaveCID, enclavePort uint32) *Vsock {
	v := &Vsock{
		EnclaveCID:  enclaveCID,
		EnclavePort: enclavePort,
	}
	v.pool = *NewVsockPool(enclaveCID, enclavePort, 10)
	return v
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
	if !res.Success {
		return nil, fmt.Errorf("Enclave returned error: %s", res.Error)
	}
	if len(res.Data.Attestation) == 0 {
		return nil, fmt.Errorf("Empty attestation document")
	}
	return res.Data.Attestation, nil
}

func (v *Vsock) SendToEnclave(req enclaveproto.Request, timeout time.Duration) (json.RawMessage, error) {
	jsonReq, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal request: %v", err)
	}

	c, err := v.pool.Get()
	if err != nil {
		return nil, err
	}
	c.conn.SetDeadline(time.Now().Add(timeout))
	// send request
	if _, err = c.conn.Write(append(jsonReq, '\n')); err != nil {
		c.conn.Close()
		return nil, fmt.Errorf("Failed to send request to enclave: %v", err)
	}
	// read response through scanner
	if !c.scanner.Scan() {
		c.conn.Close()
		if err := c.scanner.Err(); err != nil {
			return nil, fmt.Errorf("Failed to read response from enclave: %v", err)
		}
		return nil, fmt.Errorf("no response from enclave")
	}

	res := make([]byte, len(c.scanner.Bytes()))
	copy(res, c.scanner.Bytes())

	c.conn.SetDeadline(time.Time{}) // clear deadline
	v.pool.Put(c)

	return res, nil
}

func NewVsockPool(cid, port uint32, maxIdle int) *VsockPool {
	return &VsockPool{
		cid:     cid,
		port:    port,
		idle:    make([]*vsockConn, 0, maxIdle),
		maxIdle: maxIdle,
	}
}

func (p *VsockPool) Get() (*vsockConn, error) {
	p.mu.Lock()

	if len(p.idle) > 0 {
		c := p.idle[len(p.idle)-1]
		p.idle = p.idle[:len(p.idle)-1]
		p.mu.Unlock() // release lock before returning
		return c, nil
	}
	p.mu.Unlock() // release lock before dialing

	conn, err := vsock.Dial(p.cid, p.port, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to enclave vsock: %v", err)
	}
	return &vsockConn{
		conn:    conn,
		scanner: bufio.NewScanner(conn),
		encoder: json.NewEncoder(conn),
	}, nil
}

func (p *VsockPool) Put(c *vsockConn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.idle) >= p.maxIdle {
		c.conn.Close()
		return
	}
	p.idle = append(p.idle, c)
}

func (p *VsockPool) Close(c *vsockConn) {
	c.conn.Close()
}
