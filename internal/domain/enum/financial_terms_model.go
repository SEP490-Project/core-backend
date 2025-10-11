package enum

import (
	"database/sql/driver"
	"fmt"
)

// FinancialTermsModel represents the model of financial terms.
// It can be one of the following values: "FIXED", "LEVELS", or "SHARE".
type FinancialTermsModel string

const (
	FinancialTermsModelFixed  FinancialTermsModel = "FIXED"
	FinancialTermsModelLevels FinancialTermsModel = "LEVELS"
	FinancialTermsModelShare  FinancialTermsModel = "SHARE"
)

func (ftm FinancialTermsModel) IsValid() bool {
	switch ftm {
	case FinancialTermsModelFixed, FinancialTermsModelLevels, FinancialTermsModelShare:
		return true
	}
	return false
}

func (ftm *FinancialTermsModel) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan UserRole: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*ftm = FinancialTermsModel(s)
	return nil
}

func (ftm FinancialTermsModel) Value() (driver.Value, error) {
	return string(ftm), nil
}

func (ftm FinancialTermsModel) String() string { return string(ftm) }
