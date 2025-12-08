package enum

import (
	"database/sql/driver"
	"fmt"
)

type ReactionType string

const (
	ReactionTypeLike     ReactionType = "LIKE"
	ReactionTypeLove     ReactionType = "LOVE"
	ReactionTypeWow      ReactionType = "WOW"
	ReactionTypeHaha     ReactionType = "HAHA"
	ReactionTypeSad      ReactionType = "SAD"
	ReactionTypeAngry    ReactionType = "ANGRY"
	ReactionTypeThankful ReactionType = "THANKFUL"
)

func (rt ReactionType) IsValid() bool {
	switch rt {
	case ReactionTypeLike, ReactionTypeLove, ReactionTypeWow, ReactionTypeHaha, ReactionTypeSad, ReactionTypeAngry, ReactionTypeThankful:
		return true
	}
	return false
}

func (rt *ReactionType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ReactionType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*rt = ReactionType(s)
	return nil
}

func (rt ReactionType) Value() (driver.Value, error) {
	return string(rt), nil
}

func (rt ReactionType) String() string {
	return string(rt)
}
