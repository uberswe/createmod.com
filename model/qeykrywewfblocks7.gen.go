// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameQeyKryWEwfblocks7 = "QeyKryWEwfblocks7"

// QeyKryWEwfblocks7 mapped from table <QeyKryWEwfblocks7>
type QeyKryWEwfblocks7 struct {
	ID          int64  `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	Type        int32  `gorm:"column:type;not null" json:"type"`
	IP          []byte `gorm:"column:IP;not null;default:0x" json:"IP"`
	BlockedTime int64  `gorm:"column:blockedTime;not null" json:"blockedTime"`
	Reason      string `gorm:"column:reason;not null" json:"reason"`
	LastAttempt int32  `gorm:"column:lastAttempt" json:"lastAttempt"`
	BlockedHits int32  `gorm:"column:blockedHits" json:"blockedHits"`
	Expiration  int64  `gorm:"column:expiration;not null" json:"expiration"`
	Parameters  string `gorm:"column:parameters" json:"parameters"`
}

// TableName QeyKryWEwfblocks7's table name
func (*QeyKryWEwfblocks7) TableName() string {
	return TableNameQeyKryWEwfblocks7
}
