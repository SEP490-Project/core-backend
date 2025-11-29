package constant

import (
	"core-backend/pkg/utils"
	"strings"
)

type AdminConfigConst string

const (
	ConfigKeyMinimumDayBeforeContracPaymentDue AdminConfigConst = "MINIMUM_DAY_BEFORE_CONTRACT_PAYMENT_DUE"
	ConfigKeyPayOSLinkExpiry                   AdminConfigConst = "PAYOS_LINK_EXPIRY"
	ConfigKeyRepresentativeName                AdminConfigConst = "REPRESENTATIVE_NAME"
	ConfigKeyRepresentativeRole                AdminConfigConst = "REPRESENTATIVE_ROLE"
	ConfigKeyRepresentativePhone               AdminConfigConst = "REPRESENTATIVE_PHONE"
	ConfigKeyRepresentativeEmail               AdminConfigConst = "REPRESENTATIVE_EMAIL"
	ConfigKeyRepresentativeTaxNumber           AdminConfigConst = "REPRESENTATIVE_TAX_NUMBER"
	ConfigKeyRepresentativeBankName            AdminConfigConst = "REPRESENTATIVE_BANK_NAME"
	ConfigKeyRepresentativeBankAccountNumber   AdminConfigConst = "REPRESENTATIVE_BANK_ACCOUNT_NUMBER"
	ConfigKeyRepresentativeBankAccountHolder   AdminConfigConst = "REPRESENTATIVE_BANK_ACCOUNT_HOLDER"
	ConfigKeyRepresentativeCompanyAddress      AdminConfigConst = "REPRESENTATIVE_COMPANY_ADDRESS"
)

func (c AdminConfigConst) String() string { return string(c) }

func (c AdminConfigConst) StructFieldName() string {
	value := string(c)
	titleCaseValue := utils.ToTitleCase(value)
	return strings.ReplaceAll(titleCaseValue, " ", "")
}
