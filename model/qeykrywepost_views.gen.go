// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameQeyKryWEpostView = "QeyKryWEpost_views"

// QeyKryWEpostView mapped from table <QeyKryWEpost_views>
type QeyKryWEpostView struct {
	ID     int64  `gorm:"column:id;primaryKey" json:"id"`
	Type   int32  `gorm:"column:type;primaryKey" json:"type"`
	Period string `gorm:"column:period;primaryKey" json:"period"`
	Count_ int64  `gorm:"column:count;not null" json:"count"`
}

// TableName QeyKryWEpostView's table name
func (*QeyKryWEpostView) TableName() string {
	return TableNameQeyKryWEpostView
}
