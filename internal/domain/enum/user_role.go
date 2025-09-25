// Package enum defines various enumerations used across the application.
package enum

import (
	"database/sql/driver"
	"fmt"
)

type UserRole string

const (
	UserRoleAdmin          UserRole = "ADMIN"
	UserRoleMarketingStaff UserRole = "MARKETING_STAFF"
	UserRoleContentStaff   UserRole = "CONTENT_STAFF"
	UserRoleSalesStaff     UserRole = "SALES_STAFF"
	UserRoleCustomer       UserRole = "CUSTOMER"
	UserRoleBrandPartner   UserRole = "BRAND_PARTNER"
)

func (us UserRole) IsValid() bool {
	switch us {
	case UserRoleAdmin, UserRoleMarketingStaff, UserRoleContentStaff, UserRoleSalesStaff, UserRoleCustomer, UserRoleBrandPartner:
		return true
	}
	return false
}

func (us *UserRole) Scan(value any) error {
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
	*us = UserRole(s)
	return nil
}

func (us UserRole) Value() (driver.Value, error) {
	return string(us), nil
}
