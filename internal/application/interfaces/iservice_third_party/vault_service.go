package iservice_third_party

import (
	"context"
	"core-backend/config"

	vault "github.com/hashicorp/vault/api"
)

// VaultService defines the interface for HashiCorp Vault secret management operations
type VaultService interface {
	// PutSecret stores a secret in Vault's KV v2 secrets engine
	PutSecret(ctx context.Context, secretPath string, data map[string]any) error

	// GetSecret retrieves a secret from Vault's KV v2 secrets engine
	GetSecret(ctx context.Context, secretPath string) (map[string]any, error)

	// GetSecretField retrieves a specific field from a secret in Vault
	GetSecretField(ctx context.Context, secretPath, fieldName string) (string, error)

	// SecretExists checks if a secret exists at the given path
	SecretExists(ctx context.Context, secretPath string) (bool, error)

	// DeleteSecret deletes a secret from Vault
	DeleteSecret(ctx context.Context, secretPath string) error

	// Health checks if the Vault connection is healthy
	Health(ctx context.Context) error

	// GetClient returns the underlying Vault client for advanced operations
	GetClient() *vault.Client

	// StoreRSAKeys stores RSA private and public keys in Vault (convenience method)
	StoreRSAKeys(ctx context.Context, secretPath string, privateKeyPEM, publicKeyPEM string, privateKeyField, publicKeyField string) error

	// GetRSAKeys retrieves RSA private and public keys from Vault (convenience method)
	GetRSAKeys(ctx context.Context, secretPath string, privateKeyField, publicKeyField string) (privateKeyPEM, publicKeyPEM string, err error)

	// CheckRSAKeysExist checks if both RSA keys exist in Vault (convenience method)
	CheckRSAKeysExist(ctx context.Context, secretPath string, privateKeyField, publicKeyField string) (bool, error)

	// InitializeRSAKeys ensures RSA keys are stored in Vault. If keys don't exist, generates and stores them.
	InitializeRSAKeys(ctx context.Context, jwtConfig *config.JWTConfig) error
}
