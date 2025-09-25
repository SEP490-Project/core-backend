package model

// ProductType maps to table `product_type`
type ProductType struct {
	TypeID   int       `gorm:"column:typeid;primaryKey;autoIncrement"`
	Name     *string   `gorm:"column:name;size:50;unique"`
	Products []Product `gorm:"-"` // ignore during migration to prevent premature recursive AutoMigrate
}
