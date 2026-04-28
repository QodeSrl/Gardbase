package tenants

type CreateTenantRequest struct {
	// TODO: later on, implement advanced self-managed keys
	// ClientPubKey string `json:"client_public_key" binding:"required"`
}

type CreateAPIKeyRequest struct {
	Permissions []string `json:"permissions" binding:"required"`
}

type DeleteAPIKeyRequest struct {
	APIKey string `json:"api_key" binding:"required"`
}

type ListAPIKeysRequest struct {
}
