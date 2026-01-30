package tenants

type CreateTenantResponse struct {
	TenantID string `json:"tenant_id"`
	// TODO: later on, implement advanced self-managed keys
	// EncryptedMasterKey  string `json:"encrypted_master_key"`
	// EncryptedTableSalt  string `json:"encrypted_table_salt"`
	// EnclavePubKey       string `json:"enclave_public_key"`
	AttestationDocument string `json:"attestation_document"`
	// Nonce               string `json:"nonce"`
	APIKey string `json:"api_key"`
}
