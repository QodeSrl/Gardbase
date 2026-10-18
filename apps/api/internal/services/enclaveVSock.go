package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
	"uuid"

	"github.com/mdlayher/vsock"
	"github.com/qodesrl/gardbase/pkg/enclaveproto"
)

type VsockMux struct {
	wmu     *sync.Mutex
	mu      *sync.Mutex
	conn    net.Conn
	enc     *json.Encoder
	scanner *bufio.Scanner
	pending map[string]chan json.RawMessage
	cid     uint32
	port    uint32
	dead    bool
}

type Vsock struct {
	// Enclave context ID
	EnclaveCID uint32
	// Enclave listening port
	EnclavePort uint32
	// Mux for handling multiple requests over a single vsock connection
	mux *VsockMux
}

func NewVsock(enclaveCID, enclavePort uint32) (*Vsock, error) {
	mux, err := newVsockMux(enclaveCID, enclavePort)
	if err != nil {
		return nil, fmt.Errorf("Failed to create vsock mux: %v", err)
	}
	return &Vsock{
		EnclaveCID:  enclaveCID,
		EnclavePort: enclavePort,
		mux:         mux,
	}, nil
}

func newVsockMux(cid, port uint32) (*VsockMux, error) {
	conn, err := vsock.Dial(cid, port, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to enclave vsock: %v", err)
	}
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	m := &VsockMux{
		conn:    conn,
		enc:     json.NewEncoder(conn),
		scanner: scanner,
		pending: make(map[string]chan json.RawMessage),
		cid:     cid,
		port:    port,
	}
	go m.readLoop()
	return m, nil
}

func (m *VsockMux) readLoop() {
	for m.scanner.Scan() {
		raw := make([]byte, len(m.scanner.Bytes()))
		copy(raw, m.scanner.Bytes())

		var header struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(raw, &header); err != nil || header.ID == "" {
			continue
		}

		m.mu.Lock()
		ch, ok := m.pending[header.ID]
		if ok {
			delete(m.pending, header.ID)
		}
		m.mu.Unlock()

		if ok {
			ch <- raw
		}
	}

	if err := m.scanner.Err(); err != nil {
		log.Printf("Error reading from vsock: %v", err)
	}

	// connection died
	m.mu.Lock()
	m.dead = true
	for id, ch := range m.pending {
		close(ch)
		delete(m.pending, id)
	}
	m.mu.Unlock()
}

func (m *VsockMux) reconnect() error {
	conn, err := vsock.Dial(m.cid, m.port, nil)
	if err != nil {
		return fmt.Errorf("Failed to reconnect to enclave vsock: %v", err)
	}
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	m.wmu.Lock()
	m.mu.Lock()
	m.conn = conn
	m.enc = json.NewEncoder(conn)
	m.scanner = scanner
	m.dead = false
	m.mu.Unlock()
	m.wmu.Unlock()

	go m.readLoop()
	return nil
}

func (m *VsockMux) Send(req enclaveproto.Request, timeout time.Duration) (json.RawMessage, error) {
	req.ID = uuid.New().String()
	ch := make(chan json.RawMessage, 1)

	m.mu.Lock()
	if m.dead {
		m.mu.Unlock()
		if err := m.reconnect(); err != nil {
			return nil, err
		}
		m.mu.Lock()
	}
	m.pending[req.ID] = ch
	m.mu.Unlock()

	m.wmu.Lock()
	err := m.enc.Encode(req)
	m.wmu.Unlock()

	if err != nil {
		m.mu.Lock()
		delete(m.pending, req.ID)
		m.mu.Unlock()
		return nil, fmt.Errorf("Failed to send request to enclave: %v", err)
	}

	select {
	case res, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("Connection to enclave closed")
		}
		return res, nil
	case <-time.After(timeout):
		m.mu.Lock()
		delete(m.pending, req.ID)
		m.mu.Unlock()
		return nil, fmt.Errorf("Request to enclave timed out")
	}
}

func (v *Vsock) SendToEnclave(req enclaveproto.Request, timeout time.Duration) (json.RawMessage, error) {
	return v.mux.Send(req, timeout)
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
