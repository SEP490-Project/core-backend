package enum

import (
	"database/sql/driver"
	"fmt"
)

type ConfigValueType string

const (
	ConfigValueTypeString   ConfigValueType = "STRING"
	ConfigValueTypeTextArea ConfigValueType = "TEXTAREA"
	ConfigValueTypeNumber   ConfigValueType = "NUMBER"
	ConfigValueTypeBoolean  ConfigValueType = "BOOLEAN"
	ConfigValueTypeJSON     ConfigValueType = "JSON"
	ConfigValueTypeArray    ConfigValueType = "ARRAY"
	ConfigValueTypeTime     ConfigValueType = "TIME"
)

func (cvt ConfigValueType) IsValid() bool {
	switch cvt {
	case ConfigValueTypeString, ConfigValueTypeTextArea, ConfigValueTypeNumber, ConfigValueTypeBoolean, ConfigValueTypeJSON, ConfigValueTypeArray, ConfigValueTypeTime:
		return true
	}
	return false
}

func (cvt *ConfigValueType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ConfigValueType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*cvt = ConfigValueType(s)
	return nil
}

func (cvt ConfigValueType) Value() (driver.Value, error) {
	return string(cvt), nil
}

func (cvt ConfigValueType) String() string {
	return string(cvt)
}
