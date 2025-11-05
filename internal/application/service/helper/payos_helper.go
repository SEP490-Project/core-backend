package helper

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// GeneratePayOSDescription generates a meaningful payment description for PayOS
// Format: <PREFIX>-<SHORT_UUID>
// PREFIX: CON (Contract Payment) or ORD (Order Payment)
// SHORT_UUID: First 8 characters of the payment transaction UUID
//
// Examples:
//   - CON-a1b2c3d4 (Contract Payment)
//   - ORD-f5e6d7c8 (Order Payment)
//
// PayOS limits description to 9 characters for non-linked bank accounts,
// so this format ensures we stay within that limit while being meaningful.
func GeneratePayOSDescription(referenceType string, paymentTransactionID uuid.UUID) string {
	var prefix string

	// Determine prefix based on reference type
	switch strings.ToUpper(referenceType) {
	case "CONTRACT_PAYMENT":
		prefix = "CON"
	case "ORDER":
		prefix = "ORD"
	default:
		// Fallback: take first 3 letters of type or default to PAY
		if len(referenceType) >= 3 {
			prefix = strings.ToUpper(referenceType[:3])
		} else {
			prefix = "PAY"
		}
	}

	// Remove dashes from UUID
	uuidStr := strings.ReplaceAll(paymentTransactionID.String(), "-", "")

	// Compute max suffix length to stay ≤ 9 characters total (including dash)
	maxSuffixLen := 9 - (len(prefix) + 1)
	if maxSuffixLen < 0 {
		maxSuffixLen = 0
	}
	if maxSuffixLen > len(uuidStr) {
		maxSuffixLen = len(uuidStr)
	}

	shortUUID := uuidStr[:maxSuffixLen]

	return fmt.Sprintf("%s-%s", prefix, shortUUID)
}

// ExtractPaymentTransactionIDFromDescription extracts the payment transaction UUID from the PayOS description
func ExtractPaymentTransactionIDFromDescription(description string) uuid.UUID {
	// Extract UUID from description
	uuidStr := strings.Split(description, "-")[1]
	paymentTransactionID, _ := uuid.Parse(uuidStr)
	return paymentTransactionID
}
