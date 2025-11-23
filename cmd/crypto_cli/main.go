package main

import (
	"core-backend/pkg/crypto"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

// MODIFIED: Function signature and logic updated to accept a direct key string.
// getKey retrieves the encryption key based on a clear precedence:
// 1. Direct key argument (-key)
// 2. Environment variable (AES_KEY)
// 3. Key file path (-key-file)
func getKey(hexKeyArg string, keyPath string) ([]byte, error) {
	var hexKey string

	// 1. Prioritize the direct -key flag argument.
	if hexKeyArg != "" {
		hexKey = hexKeyArg
	} else if envKey := os.Getenv("AES_KEY"); envKey != "" {
		// 2. Fallback to the environment variable.
		hexKey = envKey
	} else if keyPath != "" {
		// 3. Fallback to the key file.
		keyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read key from file: %w", err)
		}
		hexKey = string(keyBytes)
	} else {
		// 4. If no key is provided, return an error.
		return nil, errors.New("a key is required; please use the -key flag, set the AES_KEY environment variable, or use the -key-file flag")
	}

	// Clean up the key in case it has surrounding whitespace
	cleanedHexKey := strings.TrimSpace(hexKey)

	key, err := hex.DecodeString(cleanedHexKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes (64 hex characters), but got %d bytes", len(key))
	}
	return key, nil
}

// generateKey creates a new 32-byte AES key and prints it in hex format.
func generateKey() {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating key: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(hex.EncodeToString(bytes))
}

func main() {
	// Subcommand for encryption
	encryptCmd := flag.NewFlagSet("encrypt", flag.ExitOnError)
	// MODIFIED: Added -key flag with a security warning in the description.
	encryptKey := encryptCmd.String("key", "", "The 64-character hex key (INSECURE: will be logged in shell history).")
	encryptKeyFile := encryptCmd.String("key-file", "", "Path to a file containing the hex-encoded key.")

	// Subcommand for decryption
	decryptCmd := flag.NewFlagSet("decrypt", flag.ExitOnError)
	// MODIFIED: Added -key flag with a security warning in the description.
	decryptKey := decryptCmd.String("key", "", "The 64-character hex key (INSECURE: will be logged in shell history).")
	decryptKeyFile := decryptCmd.String("key-file", "", "Path to a file containing the hex-encoded key.")

	// Subcommand for key generation
	generateCmd := flag.NewFlagSet("generate-key", flag.ExitOnError)

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd <command> [arguments]")
		fmt.Println("Available commands: encrypt, decrypt, generate-key")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate-key":
		generateCmd.Parse(os.Args[2:])
		generateKey()

	case "encrypt":
		encryptCmd.Parse(os.Args[2:])
		if encryptCmd.NArg() == 0 {
			fmt.Fprintln(os.Stderr, "Error: plaintext to encrypt is required.")
			encryptCmd.Usage()
			os.Exit(1)
		}
		plaintext := encryptCmd.Arg(0)

		// MODIFIED: Pass the new -key flag value to getKey
		key, err := getKey(*encryptKey, *encryptKeyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting key: %v\n", err)
			os.Exit(1)
		}

		ciphertext, err := crypto.EncryptToken(plaintext, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Encryption failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(ciphertext)

	case "decrypt":
		decryptCmd.Parse(os.Args[2:])
		if decryptCmd.NArg() == 0 {
			fmt.Fprintln(os.Stderr, "Error: ciphertext to decrypt is required.")
			decryptCmd.Usage()
			os.Exit(1)
		}
		ciphertext := decryptCmd.Arg(0)

		// MODIFIED: Pass the new -key flag value to getKey
		key, err := getKey(*decryptKey, *decryptKeyFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting key: %v\n", err)
			os.Exit(1)
		}

		plaintext, err := crypto.DecryptToken(ciphertext, key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Decryption failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(plaintext)

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		fmt.Println("Available commands: encrypt, decrypt, generate-key")
		os.Exit(1)
	}
}
