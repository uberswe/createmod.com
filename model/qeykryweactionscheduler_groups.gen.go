// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameQeyKryWEactionschedulerGroup = "QeyKryWEactionscheduler_groups"

// QeyKryWEactionschedulerGroup mapped from table <QeyKryWEactionscheduler_groups>
type QeyKryWEactionschedulerGroup struct {
	GroupID int64  `gorm:"column:group_id;primaryKey;autoIncrement:true" json:"group_id"`
	Slug    string `gorm:"column:slug;not null" json:"slug"`
}

// TableName QeyKryWEactionschedulerGroup's table name
func (*QeyKryWEactionschedulerGroup) TableName() string {
	return TableNameQeyKryWEactionschedulerGroup
}
