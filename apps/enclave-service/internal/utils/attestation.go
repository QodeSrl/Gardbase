package utils

import (
	"fmt"
	"sync"

	"github.com/hf/nsm"
	"github.com/hf/nsm/request"
)

type Attestation struct {
	Mu  sync.RWMutex
	Doc []byte
}

func RequestAttestation(nsmSession *nsm.Session, nsmPublicKeyBytes []byte) ([]byte, error) {
	req := request.Attestation{
		PublicKey: nsmPublicKeyBytes,
		Nonce:     nil,
		UserData:  nil,
	}
	res, err := nsmSession.Send(&req)
	if err != nil {
		return nil, fmt.Errorf("failed to get attestation document: %v", err)
	}
	if res.Attestation == nil || len(res.Attestation.Document) == 0 {
		return nil, fmt.Errorf("received empty attestation document from NSM")
	}
	return res.Attestation.Document, nil
}
