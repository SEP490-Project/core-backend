package constant

type ScopeOfWorkIDType string

const (
	ScopeOfWorkIDTypeAdvertise ScopeOfWorkIDType = "ADVERTISE"
	ScopeOfWorkIDTypeAffiliate ScopeOfWorkIDType = "AFFILIATE"
	ScopeOfWorkIDTypeEvent     ScopeOfWorkIDType = "EVENT"
	ScopeOfWorkIDTypeProduct   ScopeOfWorkIDType = "PRODUCT"
	ScopeOfWorkIDTypeConcept   ScopeOfWorkIDType = "CONCEPT"
)

func (s ScopeOfWorkIDType) String() string { return string(s) }

func (s ScopeOfWorkIDType) IsValid() bool {
	switch s {
	case ScopeOfWorkIDTypeAdvertise, ScopeOfWorkIDTypeAffiliate, ScopeOfWorkIDTypeEvent, ScopeOfWorkIDTypeProduct, ScopeOfWorkIDTypeConcept:
		return true
	default:
		return false
	}
}
