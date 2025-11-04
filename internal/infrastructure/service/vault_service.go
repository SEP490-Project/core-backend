package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/pkg/crypto"
	"fmt"
	"strings"

	vault "github.com/hashicorp/vault/api"
	"go.uber.org/zap"
)

// VaultService handles interactions with HashiCorp Vault for general secret management
// It supports storing and retrieving various types of secrets including API keys, tokens, and credentials
type VaultService struct {
	client *vault.Client
}

// NewVaultService initializes a new Vault client and returns a VaultService instance
func NewVaultService(vaultConfig *config.VaultConfig) (iservice_third_party.VaultService, error) {
	if vaultConfig == nil {
		return nil, fmt.Errorf("vault configuration is nil")
	}

	if !vaultConfig.Enabled {
		zap.L().Info("Vault integration is disabled")
		return nil, nil
	}

	// Validate required configuration
	if vaultConfig.Address == "" {
		return nil, fmt.Errorf("vault address is required when Vault is enabled")
	}
	if vaultConfig.Token == "" {
		return nil, fmt.Errorf("vault token is required when Vault is enabled")
	}

	// Create Vault client configuration
	vaultClientConfig := vault.DefaultConfig()
	vaultClientConfig.Address = vaultConfig.Address

	// Create the Vault client
	client, err := vault.NewClient(vaultClientConfig)
	if err != nil {
		zap.L().Error("Failed to create Vault client",
			zap.String("address", vaultConfig.Address),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	// Set the Vault token
	client.SetToken(vaultConfig.Token)

	zap.L().Info("Vault client initialized successfully",
		zap.String("address", vaultConfig.Address))

	return &VaultService{
		client: client,
	}, nil
}

// PutSecret stores a secret in Vault's KV v2 secrets engine
// secretPath: full path including mount (e.g., "secret/jwt-keys", "secret/data/jwt-keys")
// data: map of key-value pairs to store
func (v *VaultService) PutSecret(ctx context.Context, secretPath string, data map[string]any) error {
	if v == nil || v.client == nil {
		return fmt.Errorf("vault client is not initialized")
	}

	zap.L().Info("Storing secret in Vault",
		zap.String("secret_path", secretPath))

	mountPath, path := parseVaultPath(secretPath)
	kvv2 := v.client.KVv2(mountPath)
	_, err := kvv2.Put(ctx, path, data)

	if err != nil {
		zap.L().Error("Failed to store secret in Vault",
			zap.String("secret_path", secretPath),
			zap.Error(err))
		return fmt.Errorf("failed to store secret in Vault: %w", err)
	}

	zap.L().Info("Successfully stored secret in Vault",
		zap.String("secret_path", secretPath))

	return nil
}

// GetSecret retrieves a secret from Vault's KV v2 secrets engine
// secretPath: full path including mount (e.g., "secret/jwt-keys")
// Returns the secret data as a map
func (v *VaultService) GetSecret(ctx context.Context, secretPath string) (map[string]any, error) {
	if v == nil || v.client == nil {
		return nil, fmt.Errorf("vault client is not initialized")
	}

	zap.L().Debug("Retrieving secret from Vault",
		zap.String("secret_path", secretPath))

	mountPath, path := parseVaultPath(secretPath)
	kvv2 := v.client.KVv2(mountPath)
	secret, err := kvv2.Get(ctx, path)

	if err != nil {
		zap.L().Error("Failed to retrieve secret from Vault",
			zap.String("secret_path", secretPath),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve secret from Vault: %w", err)
	}

	if secret == nil || secret.Data == nil {
		zap.L().Error("No secret data found in Vault",
			zap.String("secret_path", secretPath))
		return nil, fmt.Errorf("no secret data found at path: %s", secretPath)
	}

	zap.L().Debug("Successfully retrieved secret from Vault",
		zap.String("secret_path", secretPath))

	return secret.Data, nil
}

// GetSecretField retrieves a specific field from a secret in Vault
// Useful for getting single values like API keys or tokens
func (v *VaultService) GetSecretField(ctx context.Context, secretPath, fieldName string) (string, error) {
	data, err := v.GetSecret(ctx, secretPath)
	if err != nil {
		return "", err
	}

	fieldValue, ok := data[fieldName]
	if !ok {
		return "", fmt.Errorf("field '%s' not found in secret at path: %s", fieldName, secretPath)
	}

	strValue, ok := fieldValue.(string)
	if !ok {
		return "", fmt.Errorf("field '%s' is not a string", fieldName)
	}

	return strValue, nil
}

// SecretExists checks if a secret exists at the given path
func (v *VaultService) SecretExists(ctx context.Context, secretPath string) (bool, error) {
	if v == nil || v.client == nil {
		return false, fmt.Errorf("vault client is not initialized")
	}

	zap.L().Debug("Checking if secret exists in Vault",
		zap.String("secret_path", secretPath))

	mountPath, path := parseVaultPath(secretPath)
	kvv2 := v.client.KVv2(mountPath)
	secret, err := kvv2.Get(ctx, path)

	if err != nil {
		// If the error is "secret not found", return false (not an error condition)
		if vault.ErrSecretNotFound != nil && err.Error() == vault.ErrSecretNotFound.Error() {
			return false, nil
		}
		// Other errors should be returned
		zap.L().Warn("Error checking if secret exists in Vault",
			zap.String("secret_path", secretPath),
			zap.Error(err))
		return false, err
	}

	return secret != nil && secret.Data != nil, nil
}

// DeleteSecret deletes a secret from Vault
func (v *VaultService) DeleteSecret(ctx context.Context, secretPath string) error {
	if v == nil || v.client == nil {
		return fmt.Errorf("vault client is not initialized")
	}

	zap.L().Info("Deleting secret from Vault",
		zap.String("secret_path", secretPath))

	mountPath, path := parseVaultPath(secretPath)
	kvv2 := v.client.KVv2(mountPath)
	err := kvv2.Delete(ctx, path)

	if err != nil {
		zap.L().Error("Failed to delete secret from Vault",
			zap.String("secret_path", secretPath),
			zap.Error(err))
		return fmt.Errorf("failed to delete secret from Vault: %w", err)
	}

	zap.L().Info("Successfully deleted secret from Vault",
		zap.String("secret_path", secretPath))

	return nil
}

// Health checks if the Vault connection is healthy
func (v *VaultService) Health(ctx context.Context) error {
	if v == nil || v.client == nil {
		return fmt.Errorf("vault client is not initialized")
	}

	// Check Vault health status
	health, err := v.client.Sys().HealthWithContext(ctx)
	if err != nil {
		zap.L().Error("Vault health check failed", zap.Error(err))
		return fmt.Errorf("vault health check failed: %w", err)
	}

	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	zap.L().Debug("Vault health check passed")
	return nil
}

// GetClient returns the underlying Vault client for advanced operations
// Use with caution - prefer using the higher-level methods when possible
func (v *VaultService) GetClient() *vault.Client {
	if v == nil {
		return nil
	}
	return v.client
}

// parseVaultPath extracts the mount path and secret path from a full Vault path
// Examples:
//
//	"secret/jwt-keys" -> ("secret", "jwt-keys")
//	"secret/data/jwt-keys" -> ("secret", "jwt-keys")  // KV v2 format auto-handled
func parseVaultPath(fullPath string) (mountPath, secretPath string) {
	parts := strings.Split(fullPath, "/")
	if len(parts) == 0 {
		return "", ""
	}

	mountPath = parts[0]

	// Check if the path contains "data/" (KV v2 format)
	// The KVv2 client handles this automatically, so we remove it
	if len(parts) > 2 && parts[1] == "data" {
		// Remove mount and "data" from path
		secretPath = strings.Join(parts[2:], "/")
	} else if len(parts) > 1 {
		// Regular path format (mount/path)
		secretPath = strings.Join(parts[1:], "/")
	}

	return mountPath, secretPath
}

// =============================================================================
// RSA Key Management Helper Methods (using general-purpose VaultService)
// =============================================================================

// StoreRSAKeys stores RSA private and public keys in Vault
// This is a convenience method for the common use case of storing JWT keys
func (v *VaultService) StoreRSAKeys(ctx context.Context, secretPath string, privateKeyPEM, publicKeyPEM string, privateKeyField, publicKeyField string) error {
	// Set defaults for field names
	if privateKeyField == "" {
		privateKeyField = "private_key"
	}
	if publicKeyField == "" {
		publicKeyField = "public_key"
	}

	data := map[string]any{
		privateKeyField: privateKeyPEM,
		publicKeyField:  publicKeyPEM,
	}

	return v.PutSecret(ctx, secretPath, data)
}

// GetRSAKeys retrieves RSA private and public keys from Vault
// This is a convenience method for the common use case of retrieving JWT keys
func (v *VaultService) GetRSAKeys(ctx context.Context, secretPath string, privateKeyField, publicKeyField string) (privateKeyPEM, publicKeyPEM string, err error) {
	// Set defaults for field names
	if privateKeyField == "" {
		privateKeyField = "private_key"
	}
	if publicKeyField == "" {
		publicKeyField = "public_key"
	}

	data, err := v.GetSecret(ctx, secretPath)
	if err != nil {
		return "", "", err
	}

	// Extract private key
	privateKeyRaw, ok := data[privateKeyField]
	if !ok {
		return "", "", fmt.Errorf("private key field '%s' not found in Vault secret", privateKeyField)
	}
	privateKeyPEM, ok = privateKeyRaw.(string)
	if !ok {
		return "", "", fmt.Errorf("private key is not a string")
	}

	// Extract public key
	publicKeyRaw, ok := data[publicKeyField]
	if !ok {
		return "", "", fmt.Errorf("public key field '%s' not found in Vault secret", publicKeyField)
	}
	publicKeyPEM, ok = publicKeyRaw.(string)
	if !ok {
		return "", "", fmt.Errorf("public key is not a string")
	}

	return privateKeyPEM, publicKeyPEM, nil
}

// CheckRSAKeysExist checks if both RSA keys exist in Vault
func (v *VaultService) CheckRSAKeysExist(ctx context.Context, secretPath string, privateKeyField, publicKeyField string) (bool, error) {
	// Set defaults for field names
	if privateKeyField == "" {
		privateKeyField = "private_key"
	}
	if publicKeyField == "" {
		publicKeyField = "public_key"
	}

	data, err := v.GetSecret(ctx, secretPath)
	if err != nil {
		// If secret doesn't exist, that's not an error - just return false
		if strings.Contains(err.Error(), "no secret data found") {
			return false, nil
		}
		return false, err
	}

	_, hasPrivate := data[privateKeyField]
	_, hasPublic := data[publicKeyField]

	return hasPrivate && hasPublic, nil
}

// InitializeRSAKeys ensures RSA keys are stored in Vault.
// If keys don't exist in Vault, it generates them and stores them.
// It also updates the JWT config with the keys from Vault.
func (v *VaultService) InitializeRSAKeys(ctx context.Context, jwtConfig *config.JWTConfig) error {
	if v == nil || v.client == nil {
		return fmt.Errorf("vault client is not initialized")
	}

	if jwtConfig.Vault == nil || !jwtConfig.Vault.Enabled {
		return fmt.Errorf("vault is not enabled in JWT config")
	}

	vaultCfg := jwtConfig.Vault

	// Set defaults for field names
	privateKeyField := vaultCfg.PrivateKeyField
	if privateKeyField == "" {
		privateKeyField = "private_key"
	}
	publicKeyField := vaultCfg.PublicKeyField
	if publicKeyField == "" {
		publicKeyField = "public_key"
	}

	// Check if keys already exist in Vault
	exists, err := v.CheckRSAKeysExist(ctx, vaultCfg.SecretPath, privateKeyField, publicKeyField)
	if err != nil {
		return fmt.Errorf("failed to check if RSA keys exist in Vault: %w", err)
	}

	if exists {
		zap.L().Info("RSA keys already exist in Vault",
			zap.String("secret_path", vaultCfg.SecretPath))
		var privateKeyPEM, publicKeyPEM string
		privateKeyPEM, publicKeyPEM, err = v.GetRSAKeys(ctx, vaultCfg.SecretPath, privateKeyField, publicKeyField)
		if err != nil {
			zap.L().Error("Failed to retrieve existing RSA keys from Vault",
				zap.String("secret_path", vaultCfg.SecretPath),
				zap.Error(err))
			return fmt.Errorf("failed to get RSA keys from Vault: %w", err)
		}
		jwtConfig.UpdateRSAKeys(privateKeyPEM, publicKeyPEM)
		return nil
	}

	// Keys don't exist - generate them
	zap.L().Info("RSA keys not found in Vault, generating new key pair...",
		zap.String("secret_path", vaultCfg.SecretPath))

	// Generate keys in memory using crypto package
	privateKeyPEM, publicKeyPEM, err := crypto.GenerateRSAKeyPairPEM(2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	// Store keys in Vault
	if err := v.StoreRSAKeys(ctx, vaultCfg.SecretPath, privateKeyPEM, publicKeyPEM, privateKeyField, publicKeyField); err != nil {
		return fmt.Errorf("failed to store RSA keys in Vault: %w", err)
	}

	zap.L().Info("Successfully generated and stored RSA keys in Vault",
		zap.String("secret_path", vaultCfg.SecretPath))

	// Update the JWT config with the newly generated keys
	if err := jwtConfig.UpdateRSAKeys(privateKeyPEM, publicKeyPEM); err != nil {
		return fmt.Errorf("failed to update JWT config with generated keys: %w", err)
	}

	return nil
}
