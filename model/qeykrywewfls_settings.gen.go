// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameQeyKryWEwflsSetting = "QeyKryWEwfls_settings"

// QeyKryWEwflsSetting mapped from table <QeyKryWEwfls_settings>
type QeyKryWEwflsSetting struct {
	Name     string `gorm:"column:name;primaryKey" json:"name"`
	Value    []byte `gorm:"column:value" json:"value"`
	Autoload string `gorm:"column:autoload;not null;default:yes" json:"autoload"`
}

// TableName QeyKryWEwflsSetting's table name
func (*QeyKryWEwflsSetting) TableName() string {
	return TableNameQeyKryWEwflsSetting
}
