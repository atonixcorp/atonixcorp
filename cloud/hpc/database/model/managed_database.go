package model

type ManagedDatabase struct {
	ID             string `json:"id"`
	Engine         string `json:"engine"`
	Endpoint       string `json:"endpoint"`
	Port           uint16 `json:"port"`
	DatabaseName   string `json:"database_name"`
	CredentialRef  string `json:"credential_ref"`
	TLSRequired    bool   `json:"tls_required"`
	NetworkID      string `json:"network_id"`
}
