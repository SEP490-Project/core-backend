package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"
)

type StatePayload struct {
	Nonce     string `json:"nonce"`
	Timestamp int64  `json:"timestamp"`
	Expiry    int64  `json:"expiry,omitempty"`
	// Redirect  string `json:"redirect,omitempty"`
}

// GenerateStateToken generates a signed state token using an RSA private key
func GenerateStateToken(privateKey *rsa.PrivateKey, expiry *int64, redirect string) (string, error) {
	payload := StatePayload{
		Nonce:     generateRandomString(16),
		Timestamp: time.Now().Unix(),
		Expiry:    300, // Token valid for 5 minutes
		// Redirect:  redirect,
	}
	if expiry != nil && *expiry > 0 {
		payload.Expiry = *expiry
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// Hash the payload
	hash := sha256.Sum256(payloadBytes)

	// Sign with private key
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}

	// Combine payload and signature
	tokenStruct := struct {
		Payload   string `json:"p"`
		Signature string `json:"s"`
	}{
		Payload:   base64.RawURLEncoding.EncodeToString(payloadBytes),
		Signature: base64.RawURLEncoding.EncodeToString(signature),
	}

	tokenBytes, err := json.Marshal(tokenStruct)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}

// VerifyStateToken verifies the signed state token using the RSA public key
func VerifyStateToken(publicKey *rsa.PublicKey, token string) (*StatePayload, error) {
	tokenBytes, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}

	var tokenStruct struct {
		Payload   string `json:"p"`
		Signature string `json:"s"`
	}

	if err = json.Unmarshal(tokenBytes, &tokenStruct); err != nil {
		return nil, errors.New("invalid state token format")
	}

	var payloadBytes []byte
	payloadBytes, err = base64.RawURLEncoding.DecodeString(tokenStruct.Payload)
	if err != nil {
		return nil, errors.New("invalid state token payload encoding")
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(tokenStruct.Signature)
	if err != nil {
		return nil, errors.New("invalid state token signature encoding")
	}

	// Verify signature
	hash := sha256.Sum256(payloadBytes)
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], sigBytes); err != nil {
		return nil, errors.New("invalid state token signature")
	}

	var payload StatePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, errors.New("invalid state token payload format")
	}

	// Check expiry
	if payload.Expiry != 0 {
		if time.Now().Unix() > payload.Timestamp+payload.Expiry {
			return nil, errors.New("state token expired")
		}
	}

	return &payload, nil
}

// generateRandomString creates a random base64 string of given length
func generateRandomString(length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
