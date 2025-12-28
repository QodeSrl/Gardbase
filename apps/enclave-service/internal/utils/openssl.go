package utils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"os/exec"
)

func rsaPrivateKeyToPem(privKey *rsa.PrivateKey) []byte {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privKey)

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	return privateKeyPEM
}

func DecryptWithOpenSSL(ciphertextForRecipient []byte, privKey *rsa.PrivateKey) ([]byte, error) {
	privateKeyPEM := rsaPrivateKeyToPem(privKey)

	// Write ciphertext to temp file
	cipherFile, err := os.CreateTemp("", "cipher-*.der")
	if err != nil {
		return nil, err
	}
	defer os.Remove(cipherFile.Name())
	cipherFile.Write(ciphertextForRecipient)
	cipherFile.Close()

	// Write private key to temp file
	keyFile, err := os.CreateTemp("", "key-*.pem")
	if err != nil {
		return nil, err
	}
	defer os.Remove(keyFile.Name())
	keyFile.Write(privateKeyPEM)
	keyFile.Close()

	// Run openssl cms command
	cmd := exec.Command("openssl", "cms", "-decrypt",
		"-inform", "DER",
		"-in", cipherFile.Name(),
		"-inkey", keyFile.Name())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("openssl failed: %w: %s", err, output)
	}

	return output, nil
}
