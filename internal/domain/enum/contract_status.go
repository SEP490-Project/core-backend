package enum

type ContractStatus string

const (
	ContractActive  ContractStatus = "ACTIVE"
	ContractExpired ContractStatus = "EXPIRED"
	ContractCanceled ContractStatus = "CANCELED"
)
