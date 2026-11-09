package cloud

import "atonixcorp/api/internal/models"

type CloudAccountCredentials interface {
	ProviderName() string
	ScopeID() string
	RegionName() string
	Validate() error
	Encode() ([]byte, error)
}

type AccountCredentialDecoder interface {
	DecodeStoredAccount(models.CloudAccount, []byte) (CloudAccountCredentials, error)
}
