// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import (
	"time"
)

const TableNameQeyKryWEwcCommentsSubscription = "QeyKryWEwc_comments_subscription"

// QeyKryWEwcCommentsSubscription mapped from table <QeyKryWEwc_comments_subscription>
type QeyKryWEwcCommentsSubscription struct {
	ID               int32     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	Email            string    `gorm:"column:email;not null" json:"email"`
	SubscribtionID   int32     `gorm:"column:subscribtion_id;not null" json:"subscribtion_id"`
	PostID           int32     `gorm:"column:post_id;not null" json:"post_id"`
	SubscribtionType string    `gorm:"column:subscribtion_type;not null" json:"subscribtion_type"`
	ActivationKey    string    `gorm:"column:activation_key;not null" json:"activation_key"`
	Confirm          int32     `gorm:"column:confirm" json:"confirm"`
	SubscriptionDate time.Time `gorm:"column:subscription_date;not null;default:CURRENT_TIMESTAMP" json:"subscription_date"`
	ImportedFrom     string    `gorm:"column:imported_from;not null" json:"imported_from"`
}

// TableName QeyKryWEwcCommentsSubscription's table name
func (*QeyKryWEwcCommentsSubscription) TableName() string {
	return TableNameQeyKryWEwcCommentsSubscription
}
