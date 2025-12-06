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
