// Package enum defines various enumerations used across the application.
package enum

type UserRole string

const (
	RoleAdmin          UserRole = "ADMIN"
	RoleMarketingStaff UserRole = "MARKETING_STAFF"
	RoleContentStaff   UserRole = "CONTENT_STAFF"
	RoleSalesStaff     UserRole = "SALES_STAFF"
	RoleCustomer       UserRole = "CUSTOMER"
	RoleBrandPartner   UserRole = "BRAND_PARTNER"
)
