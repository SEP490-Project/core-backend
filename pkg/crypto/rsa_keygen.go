// Package crypto provides utilities for RSA key pair generation and management.
package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"go.uber.org/zap"
)

// GenerateRSAKeyPair generates a new RSA key pair and saves to files
func GenerateRSAKeyPair(privateKeyPath, publicKeyPath string, keySize int) error {
	zap.L().Info("Generating RSA key pair",
		zap.String("private_key_path", privateKeyPath),
		zap.String("public_key_path", publicKeyPath),
		zap.Int("key_size", keySize))

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		zap.L().Error("Failed to generate RSA private key",
			zap.Int("key_size", keySize),
			zap.Error(err))
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Save private key
	if err := savePrivateKey(privateKeyPath, privateKey); err != nil {
		return err
	}

	// Save public key
	if err := savePublicKey(publicKeyPath, &privateKey.PublicKey); err != nil {
		return err
	}

	zap.L().Info("Successfully generated and saved RSA key pair",
		zap.String("private_key_path", privateKeyPath),
		zap.String("public_key_path", publicKeyPath),
		zap.Int("key_size", keySize))

	return nil
}

// GenerateRSAKeyPairPEM generates a new RSA key pair and returns PEM-encoded strings
func GenerateRSAKeyPairPEM(keySize int) (privateKeyPEM, publicKeyPEM string, err error) {
	zap.L().Debug("Generating RSA key pair in PEM format",
		zap.Int("key_size", keySize))

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		zap.L().Error("Failed to generate RSA private key",
			zap.Int("key_size", keySize),
			zap.Error(err))
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Encode private key to PEM
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	privateKeyPEM = string(pem.EncodeToMemory(privateKeyBlock))

	// Encode public key to PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		zap.L().Error("Failed to marshal public key",
			zap.Error(err))
		return "", "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicKeyPEM = string(pem.EncodeToMemory(publicKeyBlock))

	zap.L().Debug("Successfully generated RSA key pair in PEM format",
		zap.Int("key_size", keySize))

	return privateKeyPEM, publicKeyPEM, nil
}

// ReadKeyFile reads a PEM-encoded key file and returns its content as a string
func ReadKeyFile(filePath string) (string, error) {
	keyBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read key file %s: %w", filePath, err)
	}
	return string(keyBytes), nil
}

// savePrivateKey saves an RSA private key to a file in PEM format
func savePrivateKey(filePath string, privateKey *rsa.PrivateKey) error {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	privateKeyFile, err := os.Create(filePath)
	if err != nil {
		zap.L().Error("Failed to create private key file",
			zap.String("private_key_path", filePath),
			zap.Error(err))
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateKeyFile.Close()

	if err = pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		zap.L().Error("Failed to write private key to file",
			zap.String("private_key_path", filePath),
			zap.Error(err))
		return fmt.Errorf("failed to write private key: %w", err)
	}

	zap.L().Debug("Successfully saved private key",
		zap.String("private_key_path", filePath))

	return nil
}

// savePublicKey saves an RSA public key to a file in PEM format
func savePublicKey(filePath string, publicKey *rsa.PublicKey) error {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		zap.L().Error("Failed to marshal public key",
			zap.Error(err))
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	publicKeyFile, err := os.Create(filePath)
	if err != nil {
		zap.L().Error("Failed to create public key file",
			zap.String("public_key_path", filePath),
			zap.Error(err))
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicKeyFile.Close()

	if err := pem.Encode(publicKeyFile, publicKeyPEM); err != nil {
		zap.L().Error("Failed to write public key to file",
			zap.String("public_key_path", filePath),
			zap.Error(err))
		return fmt.Errorf("failed to write public key: %w", err)
	}

	zap.L().Debug("Successfully saved public key",
		zap.String("public_key_path", filePath))

	return nil
}
