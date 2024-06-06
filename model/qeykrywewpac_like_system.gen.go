// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

import (
	"time"
)

const TableNameQeyKryWEwpacLikeSystem = "QeyKryWEwpac_like_system"

// QeyKryWEwpacLikeSystem mapped from table <QeyKryWEwpac_like_system>
type QeyKryWEwpacLikeSystem struct {
	ID           int32     `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	UserID       int32     `gorm:"column:user_id;not null" json:"user_id"`
	PostID       int32     `gorm:"column:post_id;not null" json:"post_id"`
	LikeCount    int32     `gorm:"column:like_count;not null" json:"like_count"`
	DislikeCount int32     `gorm:"column:dislike_count;not null" json:"dislike_count"`
	CookieID     int32     `gorm:"column:cookie_id" json:"cookie_id"`
	UserIP       string    `gorm:"column:user_ip" json:"user_ip"`
	Time         time.Time `gorm:"column:time;not null;default:CURRENT_TIMESTAMP" json:"time"`
}

// TableName QeyKryWEwpacLikeSystem's table name
func (*QeyKryWEwpacLikeSystem) TableName() string {
	return TableNameQeyKryWEwpacLikeSystem
}
