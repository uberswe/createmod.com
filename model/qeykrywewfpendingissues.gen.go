// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameQeyKryWEwfpendingissue = "QeyKryWEwfpendingissues"

// QeyKryWEwfpendingissue mapped from table <QeyKryWEwfpendingissues>
type QeyKryWEwfpendingissue struct {
	ID          int32  `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	Time        int32  `gorm:"column:time;not null" json:"time"`
	LastUpdated int32  `gorm:"column:lastUpdated;not null" json:"lastUpdated"`
	Status      string `gorm:"column:status;not null" json:"status"`
	Type        string `gorm:"column:type;not null" json:"type"`
	Severity    int32  `gorm:"column:severity;not null" json:"severity"`
	IgnoreP     string `gorm:"column:ignoreP;not null" json:"ignoreP"`
	IgnoreC     string `gorm:"column:ignoreC;not null" json:"ignoreC"`
	ShortMsg    string `gorm:"column:shortMsg;not null" json:"shortMsg"`
	LongMsg     string `gorm:"column:longMsg" json:"longMsg"`
	Data        string `gorm:"column:data" json:"data"`
}

// TableName QeyKryWEwfpendingissue's table name
func (*QeyKryWEwfpendingissue) TableName() string {
	return TableNameQeyKryWEwfpendingissue
}
