package enum

import "fmt"

type GHNDeliveryStatus string

const (
	GHNDeliveryStatusReadyToPick GHNDeliveryStatus = "ready_to_pick"
	GHNDeliveryStatusStoring     GHNDeliveryStatus = "storing"
	GHNDeliveryStatusDelivering  GHNDeliveryStatus = "delivering"
	GHNDeliveryStatusDelivered   GHNDeliveryStatus = "delivered"
	GHNDeliveryStatusCancel      GHNDeliveryStatus = "cancel"
)

func (gds GHNDeliveryStatus) IsValid() bool {
	switch gds {
	case GHNDeliveryStatusReadyToPick, GHNDeliveryStatusStoring, GHNDeliveryStatusDelivering, GHNDeliveryStatusDelivered, GHNDeliveryStatusCancel:
		return true
	}
	return false
}

func (gds *GHNDeliveryStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan GHNDeliveryStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*gds = GHNDeliveryStatus(s)
	return nil
}

func (gds GHNDeliveryStatus) String() string {
	return string(gds)
}
