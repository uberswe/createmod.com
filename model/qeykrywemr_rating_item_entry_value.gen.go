// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameQeyKryWEmrRatingItemEntryValue = "QeyKryWEmr_rating_item_entry_value"

// QeyKryWEmrRatingItemEntryValue mapped from table <QeyKryWEmr_rating_item_entry_value>
type QeyKryWEmrRatingItemEntryValue struct {
	RatingItemEntryValueID int64 `gorm:"column:rating_item_entry_value_id;primaryKey;autoIncrement:true" json:"rating_item_entry_value_id"`
	RatingItemEntryID      int64 `gorm:"column:rating_item_entry_id;not null" json:"rating_item_entry_id"`
	RatingItemID           int64 `gorm:"column:rating_item_id;not null" json:"rating_item_id"`
	Value                  int32 `gorm:"column:value;not null" json:"value"`
}

// TableName QeyKryWEmrRatingItemEntryValue's table name
func (*QeyKryWEmrRatingItemEntryValue) TableName() string {
	return TableNameQeyKryWEmrRatingItemEntryValue
}
