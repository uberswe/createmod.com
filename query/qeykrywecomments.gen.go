// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package query

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"gorm.io/gen"
	"gorm.io/gen/field"

	"gorm.io/plugin/dbresolver"

	"createmod/model"
)

func newQeyKryWEcomment(db *gorm.DB, opts ...gen.DOOption) qeyKryWEcomment {
	_qeyKryWEcomment := qeyKryWEcomment{}

	_qeyKryWEcomment.qeyKryWEcommentDo.UseDB(db, opts...)
	_qeyKryWEcomment.qeyKryWEcommentDo.UseModel(&model.QeyKryWEcomment{})

	tableName := _qeyKryWEcomment.qeyKryWEcommentDo.TableName()
	_qeyKryWEcomment.ALL = field.NewAsterisk(tableName)
	_qeyKryWEcomment.CommentID = field.NewInt64(tableName, "comment_ID")
	_qeyKryWEcomment.CommentPostID = field.NewInt64(tableName, "comment_post_ID")
	_qeyKryWEcomment.CommentAuthor = field.NewString(tableName, "comment_author")
	_qeyKryWEcomment.CommentAuthorEmail = field.NewString(tableName, "comment_author_email")
	_qeyKryWEcomment.CommentAuthorURL = field.NewString(tableName, "comment_author_url")
	_qeyKryWEcomment.CommentAuthorIP = field.NewString(tableName, "comment_author_IP")
	_qeyKryWEcomment.CommentDate = field.NewTime(tableName, "comment_date")
	_qeyKryWEcomment.CommentDateGmt = field.NewTime(tableName, "comment_date_gmt")
	_qeyKryWEcomment.CommentContent = field.NewString(tableName, "comment_content")
	_qeyKryWEcomment.CommentKarma = field.NewInt32(tableName, "comment_karma")
	_qeyKryWEcomment.CommentApproved = field.NewString(tableName, "comment_approved")
	_qeyKryWEcomment.CommentAgent = field.NewString(tableName, "comment_agent")
	_qeyKryWEcomment.CommentType = field.NewString(tableName, "comment_type")
	_qeyKryWEcomment.CommentParent = field.NewInt64(tableName, "comment_parent")
	_qeyKryWEcomment.UserID = field.NewInt64(tableName, "user_id")

	_qeyKryWEcomment.fillFieldMap()

	return _qeyKryWEcomment
}

type qeyKryWEcomment struct {
	qeyKryWEcommentDo

	ALL                field.Asterisk
	CommentID          field.Int64
	CommentPostID      field.Int64
	CommentAuthor      field.String
	CommentAuthorEmail field.String
	CommentAuthorURL   field.String
	CommentAuthorIP    field.String
	CommentDate        field.Time
	CommentDateGmt     field.Time
	CommentContent     field.String
	CommentKarma       field.Int32
	CommentApproved    field.String
	CommentAgent       field.String
	CommentType        field.String
	CommentParent      field.Int64
	UserID             field.Int64

	fieldMap map[string]field.Expr
}

func (q qeyKryWEcomment) Table(newTableName string) *qeyKryWEcomment {
	q.qeyKryWEcommentDo.UseTable(newTableName)
	return q.updateTableName(newTableName)
}

func (q qeyKryWEcomment) As(alias string) *qeyKryWEcomment {
	q.qeyKryWEcommentDo.DO = *(q.qeyKryWEcommentDo.As(alias).(*gen.DO))
	return q.updateTableName(alias)
}

func (q *qeyKryWEcomment) updateTableName(table string) *qeyKryWEcomment {
	q.ALL = field.NewAsterisk(table)
	q.CommentID = field.NewInt64(table, "comment_ID")
	q.CommentPostID = field.NewInt64(table, "comment_post_ID")
	q.CommentAuthor = field.NewString(table, "comment_author")
	q.CommentAuthorEmail = field.NewString(table, "comment_author_email")
	q.CommentAuthorURL = field.NewString(table, "comment_author_url")
	q.CommentAuthorIP = field.NewString(table, "comment_author_IP")
	q.CommentDate = field.NewTime(table, "comment_date")
	q.CommentDateGmt = field.NewTime(table, "comment_date_gmt")
	q.CommentContent = field.NewString(table, "comment_content")
	q.CommentKarma = field.NewInt32(table, "comment_karma")
	q.CommentApproved = field.NewString(table, "comment_approved")
	q.CommentAgent = field.NewString(table, "comment_agent")
	q.CommentType = field.NewString(table, "comment_type")
	q.CommentParent = field.NewInt64(table, "comment_parent")
	q.UserID = field.NewInt64(table, "user_id")

	q.fillFieldMap()

	return q
}

func (q *qeyKryWEcomment) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := q.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (q *qeyKryWEcomment) fillFieldMap() {
	q.fieldMap = make(map[string]field.Expr, 15)
	q.fieldMap["comment_ID"] = q.CommentID
	q.fieldMap["comment_post_ID"] = q.CommentPostID
	q.fieldMap["comment_author"] = q.CommentAuthor
	q.fieldMap["comment_author_email"] = q.CommentAuthorEmail
	q.fieldMap["comment_author_url"] = q.CommentAuthorURL
	q.fieldMap["comment_author_IP"] = q.CommentAuthorIP
	q.fieldMap["comment_date"] = q.CommentDate
	q.fieldMap["comment_date_gmt"] = q.CommentDateGmt
	q.fieldMap["comment_content"] = q.CommentContent
	q.fieldMap["comment_karma"] = q.CommentKarma
	q.fieldMap["comment_approved"] = q.CommentApproved
	q.fieldMap["comment_agent"] = q.CommentAgent
	q.fieldMap["comment_type"] = q.CommentType
	q.fieldMap["comment_parent"] = q.CommentParent
	q.fieldMap["user_id"] = q.UserID
}

func (q qeyKryWEcomment) clone(db *gorm.DB) qeyKryWEcomment {
	q.qeyKryWEcommentDo.ReplaceConnPool(db.Statement.ConnPool)
	return q
}

func (q qeyKryWEcomment) replaceDB(db *gorm.DB) qeyKryWEcomment {
	q.qeyKryWEcommentDo.ReplaceDB(db)
	return q
}

type qeyKryWEcommentDo struct{ gen.DO }

type IQeyKryWEcommentDo interface {
	gen.SubQuery
	Debug() IQeyKryWEcommentDo
	WithContext(ctx context.Context) IQeyKryWEcommentDo
	WithResult(fc func(tx gen.Dao)) gen.ResultInfo
	ReplaceDB(db *gorm.DB)
	ReadDB() IQeyKryWEcommentDo
	WriteDB() IQeyKryWEcommentDo
	As(alias string) gen.Dao
	Session(config *gorm.Session) IQeyKryWEcommentDo
	Columns(cols ...field.Expr) gen.Columns
	Clauses(conds ...clause.Expression) IQeyKryWEcommentDo
	Not(conds ...gen.Condition) IQeyKryWEcommentDo
	Or(conds ...gen.Condition) IQeyKryWEcommentDo
	Select(conds ...field.Expr) IQeyKryWEcommentDo
	Where(conds ...gen.Condition) IQeyKryWEcommentDo
	Order(conds ...field.Expr) IQeyKryWEcommentDo
	Distinct(cols ...field.Expr) IQeyKryWEcommentDo
	Omit(cols ...field.Expr) IQeyKryWEcommentDo
	Join(table schema.Tabler, on ...field.Expr) IQeyKryWEcommentDo
	LeftJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEcommentDo
	RightJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEcommentDo
	Group(cols ...field.Expr) IQeyKryWEcommentDo
	Having(conds ...gen.Condition) IQeyKryWEcommentDo
	Limit(limit int) IQeyKryWEcommentDo
	Offset(offset int) IQeyKryWEcommentDo
	Count() (count int64, err error)
	Scopes(funcs ...func(gen.Dao) gen.Dao) IQeyKryWEcommentDo
	Unscoped() IQeyKryWEcommentDo
	Create(values ...*model.QeyKryWEcomment) error
	CreateInBatches(values []*model.QeyKryWEcomment, batchSize int) error
	Save(values ...*model.QeyKryWEcomment) error
	First() (*model.QeyKryWEcomment, error)
	Take() (*model.QeyKryWEcomment, error)
	Last() (*model.QeyKryWEcomment, error)
	Find() ([]*model.QeyKryWEcomment, error)
	FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.QeyKryWEcomment, err error)
	FindInBatches(result *[]*model.QeyKryWEcomment, batchSize int, fc func(tx gen.Dao, batch int) error) error
	Pluck(column field.Expr, dest interface{}) error
	Delete(...*model.QeyKryWEcomment) (info gen.ResultInfo, err error)
	Update(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	Updates(value interface{}) (info gen.ResultInfo, err error)
	UpdateColumn(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateColumnSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	UpdateColumns(value interface{}) (info gen.ResultInfo, err error)
	UpdateFrom(q gen.SubQuery) gen.Dao
	Attrs(attrs ...field.AssignExpr) IQeyKryWEcommentDo
	Assign(attrs ...field.AssignExpr) IQeyKryWEcommentDo
	Joins(fields ...field.RelationField) IQeyKryWEcommentDo
	Preload(fields ...field.RelationField) IQeyKryWEcommentDo
	FirstOrInit() (*model.QeyKryWEcomment, error)
	FirstOrCreate() (*model.QeyKryWEcomment, error)
	FindByPage(offset int, limit int) (result []*model.QeyKryWEcomment, count int64, err error)
	ScanByPage(result interface{}, offset int, limit int) (count int64, err error)
	Scan(result interface{}) (err error)
	Returning(value interface{}, columns ...string) IQeyKryWEcommentDo
	UnderlyingDB() *gorm.DB
	schema.Tabler
}

func (q qeyKryWEcommentDo) Debug() IQeyKryWEcommentDo {
	return q.withDO(q.DO.Debug())
}

func (q qeyKryWEcommentDo) WithContext(ctx context.Context) IQeyKryWEcommentDo {
	return q.withDO(q.DO.WithContext(ctx))
}

func (q qeyKryWEcommentDo) ReadDB() IQeyKryWEcommentDo {
	return q.Clauses(dbresolver.Read)
}

func (q qeyKryWEcommentDo) WriteDB() IQeyKryWEcommentDo {
	return q.Clauses(dbresolver.Write)
}

func (q qeyKryWEcommentDo) Session(config *gorm.Session) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Session(config))
}

func (q qeyKryWEcommentDo) Clauses(conds ...clause.Expression) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Clauses(conds...))
}

func (q qeyKryWEcommentDo) Returning(value interface{}, columns ...string) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Returning(value, columns...))
}

func (q qeyKryWEcommentDo) Not(conds ...gen.Condition) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Not(conds...))
}

func (q qeyKryWEcommentDo) Or(conds ...gen.Condition) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Or(conds...))
}

func (q qeyKryWEcommentDo) Select(conds ...field.Expr) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Select(conds...))
}

func (q qeyKryWEcommentDo) Where(conds ...gen.Condition) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Where(conds...))
}

func (q qeyKryWEcommentDo) Order(conds ...field.Expr) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Order(conds...))
}

func (q qeyKryWEcommentDo) Distinct(cols ...field.Expr) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Distinct(cols...))
}

func (q qeyKryWEcommentDo) Omit(cols ...field.Expr) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Omit(cols...))
}

func (q qeyKryWEcommentDo) Join(table schema.Tabler, on ...field.Expr) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Join(table, on...))
}

func (q qeyKryWEcommentDo) LeftJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEcommentDo {
	return q.withDO(q.DO.LeftJoin(table, on...))
}

func (q qeyKryWEcommentDo) RightJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEcommentDo {
	return q.withDO(q.DO.RightJoin(table, on...))
}

func (q qeyKryWEcommentDo) Group(cols ...field.Expr) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Group(cols...))
}

func (q qeyKryWEcommentDo) Having(conds ...gen.Condition) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Having(conds...))
}

func (q qeyKryWEcommentDo) Limit(limit int) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Limit(limit))
}

func (q qeyKryWEcommentDo) Offset(offset int) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Offset(offset))
}

func (q qeyKryWEcommentDo) Scopes(funcs ...func(gen.Dao) gen.Dao) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Scopes(funcs...))
}

func (q qeyKryWEcommentDo) Unscoped() IQeyKryWEcommentDo {
	return q.withDO(q.DO.Unscoped())
}

func (q qeyKryWEcommentDo) Create(values ...*model.QeyKryWEcomment) error {
	if len(values) == 0 {
		return nil
	}
	return q.DO.Create(values)
}

func (q qeyKryWEcommentDo) CreateInBatches(values []*model.QeyKryWEcomment, batchSize int) error {
	return q.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (q qeyKryWEcommentDo) Save(values ...*model.QeyKryWEcomment) error {
	if len(values) == 0 {
		return nil
	}
	return q.DO.Save(values)
}

func (q qeyKryWEcommentDo) First() (*model.QeyKryWEcomment, error) {
	if result, err := q.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEcomment), nil
	}
}

func (q qeyKryWEcommentDo) Take() (*model.QeyKryWEcomment, error) {
	if result, err := q.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEcomment), nil
	}
}

func (q qeyKryWEcommentDo) Last() (*model.QeyKryWEcomment, error) {
	if result, err := q.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEcomment), nil
	}
}

func (q qeyKryWEcommentDo) Find() ([]*model.QeyKryWEcomment, error) {
	result, err := q.DO.Find()
	return result.([]*model.QeyKryWEcomment), err
}

func (q qeyKryWEcommentDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.QeyKryWEcomment, err error) {
	buf := make([]*model.QeyKryWEcomment, 0, batchSize)
	err = q.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (q qeyKryWEcommentDo) FindInBatches(result *[]*model.QeyKryWEcomment, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return q.DO.FindInBatches(result, batchSize, fc)
}

func (q qeyKryWEcommentDo) Attrs(attrs ...field.AssignExpr) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Attrs(attrs...))
}

func (q qeyKryWEcommentDo) Assign(attrs ...field.AssignExpr) IQeyKryWEcommentDo {
	return q.withDO(q.DO.Assign(attrs...))
}

func (q qeyKryWEcommentDo) Joins(fields ...field.RelationField) IQeyKryWEcommentDo {
	for _, _f := range fields {
		q = *q.withDO(q.DO.Joins(_f))
	}
	return &q
}

func (q qeyKryWEcommentDo) Preload(fields ...field.RelationField) IQeyKryWEcommentDo {
	for _, _f := range fields {
		q = *q.withDO(q.DO.Preload(_f))
	}
	return &q
}

func (q qeyKryWEcommentDo) FirstOrInit() (*model.QeyKryWEcomment, error) {
	if result, err := q.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEcomment), nil
	}
}

func (q qeyKryWEcommentDo) FirstOrCreate() (*model.QeyKryWEcomment, error) {
	if result, err := q.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEcomment), nil
	}
}

func (q qeyKryWEcommentDo) FindByPage(offset int, limit int) (result []*model.QeyKryWEcomment, count int64, err error) {
	result, err = q.Offset(offset).Limit(limit).Find()
	if err != nil {
		return
	}

	if size := len(result); 0 < limit && 0 < size && size < limit {
		count = int64(size + offset)
		return
	}

	count, err = q.Offset(-1).Limit(-1).Count()
	return
}

func (q qeyKryWEcommentDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = q.Count()
	if err != nil {
		return
	}

	err = q.Offset(offset).Limit(limit).Scan(result)
	return
}

func (q qeyKryWEcommentDo) Scan(result interface{}) (err error) {
	return q.DO.Scan(result)
}

func (q qeyKryWEcommentDo) Delete(models ...*model.QeyKryWEcomment) (result gen.ResultInfo, err error) {
	return q.DO.Delete(models)
}

func (q *qeyKryWEcommentDo) withDO(do gen.Dao) *qeyKryWEcommentDo {
	q.DO = *do.(*gen.DO)
	return q
}
