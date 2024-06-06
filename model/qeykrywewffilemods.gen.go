// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameQeyKryWEwffilemod = "QeyKryWEwffilemods"

// QeyKryWEwffilemod mapped from table <QeyKryWEwffilemods>
type QeyKryWEwffilemod struct {
	FilenameMD5        []byte `gorm:"column:filenameMD5;primaryKey" json:"filenameMD5"`
	Filename           string `gorm:"column:filename;not null" json:"filename"`
	RealPath           string `gorm:"column:real_path;not null" json:"real_path"`
	KnownFile          int32  `gorm:"column:knownFile;not null" json:"knownFile"`
	OldMD5             []byte `gorm:"column:oldMD5;not null" json:"oldMD5"`
	NewMD5             []byte `gorm:"column:newMD5;not null" json:"newMD5"`
	SHAC               []byte `gorm:"column:SHAC;not null;default:0x" json:"SHAC"`
	StoppedOnSignature string `gorm:"column:stoppedOnSignature;not null" json:"stoppedOnSignature"`
	StoppedOnPosition  int32  `gorm:"column:stoppedOnPosition;not null" json:"stoppedOnPosition"`
	IsSafeFile         string `gorm:"column:isSafeFile;not null;default:?" json:"isSafeFile"`
}

// TableName QeyKryWEwffilemod's table name
func (*QeyKryWEwffilemod) TableName() string {
	return TableNameQeyKryWEwffilemod
}
