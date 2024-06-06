// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameQeyKryWEisInvertedIndex = "QeyKryWEis_inverted_index"

// QeyKryWEisInvertedIndex mapped from table <QeyKryWEis_inverted_index>
type QeyKryWEisInvertedIndex struct {
	PostID            int64  `gorm:"column:post_id;primaryKey" json:"post_id"`
	Term              string `gorm:"column:term;primaryKey;default:0" json:"term"`
	TermReverse       string `gorm:"column:term_reverse;not null;default:0" json:"term_reverse"`
	Score             int32  `gorm:"column:score;not null" json:"score"`
	Title             int32  `gorm:"column:title;not null" json:"title"`
	Content           int32  `gorm:"column:content;not null" json:"content"`
	Excerpt           int32  `gorm:"column:excerpt;not null" json:"excerpt"`
	Comment           int32  `gorm:"column:comment;not null" json:"comment"`
	Author            int32  `gorm:"column:author;not null" json:"author"`
	Category          int32  `gorm:"column:category;not null" json:"category"`
	Tag               int32  `gorm:"column:tag;not null" json:"tag"`
	Taxonomy          int32  `gorm:"column:taxonomy;not null" json:"taxonomy"`
	Customfield       int32  `gorm:"column:customfield;not null" json:"customfield"`
	TaxonomyDetail    string `gorm:"column:taxonomy_detail;not null" json:"taxonomy_detail"`
	CustomfieldDetail string `gorm:"column:customfield_detail;not null" json:"customfield_detail"`
	Type              string `gorm:"column:type;not null;default:post" json:"type"`
	Lang              string `gorm:"column:lang;not null;default:post" json:"lang"`
}

// TableName QeyKryWEisInvertedIndex's table name
func (*QeyKryWEisInvertedIndex) TableName() string {
	return TableNameQeyKryWEisInvertedIndex
}
