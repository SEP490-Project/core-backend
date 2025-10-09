package enum

import (
	"database/sql/driver"
	"fmt"
)

type TaskType string

const (
	TaskTypeProduct TaskType = "PRODUCT"
	TaskTypeContent TaskType = "CONTENT"
	TaskTypeEvent   TaskType = "EVENT"
	TaskTypeOther   TaskType = "OTHER"
)

func (tt TaskType) IsValid() bool {
	switch tt {
	case TaskTypeProduct, TaskTypeContent, TaskTypeEvent, TaskTypeOther:
		return true
	}
	return false
}

func (tt *TaskType) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan TaskType: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*tt = TaskType(s)
	return nil
}

func (tt TaskType) Value() (driver.Value, error) {
	return string(tt), nil
}

func (tt TaskType) String() string { return string(tt) }
