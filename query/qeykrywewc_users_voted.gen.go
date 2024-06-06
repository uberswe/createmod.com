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

func newQeyKryWEwcUsersVoted(db *gorm.DB, opts ...gen.DOOption) qeyKryWEwcUsersVoted {
	_qeyKryWEwcUsersVoted := qeyKryWEwcUsersVoted{}

	_qeyKryWEwcUsersVoted.qeyKryWEwcUsersVotedDo.UseDB(db, opts...)
	_qeyKryWEwcUsersVoted.qeyKryWEwcUsersVotedDo.UseModel(&model.QeyKryWEwcUsersVoted{})

	tableName := _qeyKryWEwcUsersVoted.qeyKryWEwcUsersVotedDo.TableName()
	_qeyKryWEwcUsersVoted.ALL = field.NewAsterisk(tableName)
	_qeyKryWEwcUsersVoted.ID = field.NewInt32(tableName, "id")
	_qeyKryWEwcUsersVoted.UserID = field.NewString(tableName, "user_id")
	_qeyKryWEwcUsersVoted.CommentID = field.NewInt32(tableName, "comment_id")
	_qeyKryWEwcUsersVoted.VoteType = field.NewInt32(tableName, "vote_type")
	_qeyKryWEwcUsersVoted.IsGuest = field.NewBool(tableName, "is_guest")
	_qeyKryWEwcUsersVoted.PostID = field.NewInt64(tableName, "post_id")
	_qeyKryWEwcUsersVoted.Date = field.NewInt32(tableName, "date")

	_qeyKryWEwcUsersVoted.fillFieldMap()

	return _qeyKryWEwcUsersVoted
}

type qeyKryWEwcUsersVoted struct {
	qeyKryWEwcUsersVotedDo

	ALL       field.Asterisk
	ID        field.Int32
	UserID    field.String
	CommentID field.Int32
	VoteType  field.Int32
	IsGuest   field.Bool
	PostID    field.Int64
	Date      field.Int32

	fieldMap map[string]field.Expr
}

func (q qeyKryWEwcUsersVoted) Table(newTableName string) *qeyKryWEwcUsersVoted {
	q.qeyKryWEwcUsersVotedDo.UseTable(newTableName)
	return q.updateTableName(newTableName)
}

func (q qeyKryWEwcUsersVoted) As(alias string) *qeyKryWEwcUsersVoted {
	q.qeyKryWEwcUsersVotedDo.DO = *(q.qeyKryWEwcUsersVotedDo.As(alias).(*gen.DO))
	return q.updateTableName(alias)
}

func (q *qeyKryWEwcUsersVoted) updateTableName(table string) *qeyKryWEwcUsersVoted {
	q.ALL = field.NewAsterisk(table)
	q.ID = field.NewInt32(table, "id")
	q.UserID = field.NewString(table, "user_id")
	q.CommentID = field.NewInt32(table, "comment_id")
	q.VoteType = field.NewInt32(table, "vote_type")
	q.IsGuest = field.NewBool(table, "is_guest")
	q.PostID = field.NewInt64(table, "post_id")
	q.Date = field.NewInt32(table, "date")

	q.fillFieldMap()

	return q
}

func (q *qeyKryWEwcUsersVoted) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := q.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (q *qeyKryWEwcUsersVoted) fillFieldMap() {
	q.fieldMap = make(map[string]field.Expr, 7)
	q.fieldMap["id"] = q.ID
	q.fieldMap["user_id"] = q.UserID
	q.fieldMap["comment_id"] = q.CommentID
	q.fieldMap["vote_type"] = q.VoteType
	q.fieldMap["is_guest"] = q.IsGuest
	q.fieldMap["post_id"] = q.PostID
	q.fieldMap["date"] = q.Date
}

func (q qeyKryWEwcUsersVoted) clone(db *gorm.DB) qeyKryWEwcUsersVoted {
	q.qeyKryWEwcUsersVotedDo.ReplaceConnPool(db.Statement.ConnPool)
	return q
}

func (q qeyKryWEwcUsersVoted) replaceDB(db *gorm.DB) qeyKryWEwcUsersVoted {
	q.qeyKryWEwcUsersVotedDo.ReplaceDB(db)
	return q
}

type qeyKryWEwcUsersVotedDo struct{ gen.DO }

type IQeyKryWEwcUsersVotedDo interface {
	gen.SubQuery
	Debug() IQeyKryWEwcUsersVotedDo
	WithContext(ctx context.Context) IQeyKryWEwcUsersVotedDo
	WithResult(fc func(tx gen.Dao)) gen.ResultInfo
	ReplaceDB(db *gorm.DB)
	ReadDB() IQeyKryWEwcUsersVotedDo
	WriteDB() IQeyKryWEwcUsersVotedDo
	As(alias string) gen.Dao
	Session(config *gorm.Session) IQeyKryWEwcUsersVotedDo
	Columns(cols ...field.Expr) gen.Columns
	Clauses(conds ...clause.Expression) IQeyKryWEwcUsersVotedDo
	Not(conds ...gen.Condition) IQeyKryWEwcUsersVotedDo
	Or(conds ...gen.Condition) IQeyKryWEwcUsersVotedDo
	Select(conds ...field.Expr) IQeyKryWEwcUsersVotedDo
	Where(conds ...gen.Condition) IQeyKryWEwcUsersVotedDo
	Order(conds ...field.Expr) IQeyKryWEwcUsersVotedDo
	Distinct(cols ...field.Expr) IQeyKryWEwcUsersVotedDo
	Omit(cols ...field.Expr) IQeyKryWEwcUsersVotedDo
	Join(table schema.Tabler, on ...field.Expr) IQeyKryWEwcUsersVotedDo
	LeftJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEwcUsersVotedDo
	RightJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEwcUsersVotedDo
	Group(cols ...field.Expr) IQeyKryWEwcUsersVotedDo
	Having(conds ...gen.Condition) IQeyKryWEwcUsersVotedDo
	Limit(limit int) IQeyKryWEwcUsersVotedDo
	Offset(offset int) IQeyKryWEwcUsersVotedDo
	Count() (count int64, err error)
	Scopes(funcs ...func(gen.Dao) gen.Dao) IQeyKryWEwcUsersVotedDo
	Unscoped() IQeyKryWEwcUsersVotedDo
	Create(values ...*model.QeyKryWEwcUsersVoted) error
	CreateInBatches(values []*model.QeyKryWEwcUsersVoted, batchSize int) error
	Save(values ...*model.QeyKryWEwcUsersVoted) error
	First() (*model.QeyKryWEwcUsersVoted, error)
	Take() (*model.QeyKryWEwcUsersVoted, error)
	Last() (*model.QeyKryWEwcUsersVoted, error)
	Find() ([]*model.QeyKryWEwcUsersVoted, error)
	FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.QeyKryWEwcUsersVoted, err error)
	FindInBatches(result *[]*model.QeyKryWEwcUsersVoted, batchSize int, fc func(tx gen.Dao, batch int) error) error
	Pluck(column field.Expr, dest interface{}) error
	Delete(...*model.QeyKryWEwcUsersVoted) (info gen.ResultInfo, err error)
	Update(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	Updates(value interface{}) (info gen.ResultInfo, err error)
	UpdateColumn(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateColumnSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	UpdateColumns(value interface{}) (info gen.ResultInfo, err error)
	UpdateFrom(q gen.SubQuery) gen.Dao
	Attrs(attrs ...field.AssignExpr) IQeyKryWEwcUsersVotedDo
	Assign(attrs ...field.AssignExpr) IQeyKryWEwcUsersVotedDo
	Joins(fields ...field.RelationField) IQeyKryWEwcUsersVotedDo
	Preload(fields ...field.RelationField) IQeyKryWEwcUsersVotedDo
	FirstOrInit() (*model.QeyKryWEwcUsersVoted, error)
	FirstOrCreate() (*model.QeyKryWEwcUsersVoted, error)
	FindByPage(offset int, limit int) (result []*model.QeyKryWEwcUsersVoted, count int64, err error)
	ScanByPage(result interface{}, offset int, limit int) (count int64, err error)
	Scan(result interface{}) (err error)
	Returning(value interface{}, columns ...string) IQeyKryWEwcUsersVotedDo
	UnderlyingDB() *gorm.DB
	schema.Tabler
}

func (q qeyKryWEwcUsersVotedDo) Debug() IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Debug())
}

func (q qeyKryWEwcUsersVotedDo) WithContext(ctx context.Context) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.WithContext(ctx))
}

func (q qeyKryWEwcUsersVotedDo) ReadDB() IQeyKryWEwcUsersVotedDo {
	return q.Clauses(dbresolver.Read)
}

func (q qeyKryWEwcUsersVotedDo) WriteDB() IQeyKryWEwcUsersVotedDo {
	return q.Clauses(dbresolver.Write)
}

func (q qeyKryWEwcUsersVotedDo) Session(config *gorm.Session) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Session(config))
}

func (q qeyKryWEwcUsersVotedDo) Clauses(conds ...clause.Expression) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Clauses(conds...))
}

func (q qeyKryWEwcUsersVotedDo) Returning(value interface{}, columns ...string) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Returning(value, columns...))
}

func (q qeyKryWEwcUsersVotedDo) Not(conds ...gen.Condition) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Not(conds...))
}

func (q qeyKryWEwcUsersVotedDo) Or(conds ...gen.Condition) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Or(conds...))
}

func (q qeyKryWEwcUsersVotedDo) Select(conds ...field.Expr) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Select(conds...))
}

func (q qeyKryWEwcUsersVotedDo) Where(conds ...gen.Condition) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Where(conds...))
}

func (q qeyKryWEwcUsersVotedDo) Order(conds ...field.Expr) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Order(conds...))
}

func (q qeyKryWEwcUsersVotedDo) Distinct(cols ...field.Expr) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Distinct(cols...))
}

func (q qeyKryWEwcUsersVotedDo) Omit(cols ...field.Expr) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Omit(cols...))
}

func (q qeyKryWEwcUsersVotedDo) Join(table schema.Tabler, on ...field.Expr) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Join(table, on...))
}

func (q qeyKryWEwcUsersVotedDo) LeftJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.LeftJoin(table, on...))
}

func (q qeyKryWEwcUsersVotedDo) RightJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.RightJoin(table, on...))
}

func (q qeyKryWEwcUsersVotedDo) Group(cols ...field.Expr) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Group(cols...))
}

func (q qeyKryWEwcUsersVotedDo) Having(conds ...gen.Condition) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Having(conds...))
}

func (q qeyKryWEwcUsersVotedDo) Limit(limit int) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Limit(limit))
}

func (q qeyKryWEwcUsersVotedDo) Offset(offset int) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Offset(offset))
}

func (q qeyKryWEwcUsersVotedDo) Scopes(funcs ...func(gen.Dao) gen.Dao) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Scopes(funcs...))
}

func (q qeyKryWEwcUsersVotedDo) Unscoped() IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Unscoped())
}

func (q qeyKryWEwcUsersVotedDo) Create(values ...*model.QeyKryWEwcUsersVoted) error {
	if len(values) == 0 {
		return nil
	}
	return q.DO.Create(values)
}

func (q qeyKryWEwcUsersVotedDo) CreateInBatches(values []*model.QeyKryWEwcUsersVoted, batchSize int) error {
	return q.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (q qeyKryWEwcUsersVotedDo) Save(values ...*model.QeyKryWEwcUsersVoted) error {
	if len(values) == 0 {
		return nil
	}
	return q.DO.Save(values)
}

func (q qeyKryWEwcUsersVotedDo) First() (*model.QeyKryWEwcUsersVoted, error) {
	if result, err := q.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwcUsersVoted), nil
	}
}

func (q qeyKryWEwcUsersVotedDo) Take() (*model.QeyKryWEwcUsersVoted, error) {
	if result, err := q.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwcUsersVoted), nil
	}
}

func (q qeyKryWEwcUsersVotedDo) Last() (*model.QeyKryWEwcUsersVoted, error) {
	if result, err := q.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwcUsersVoted), nil
	}
}

func (q qeyKryWEwcUsersVotedDo) Find() ([]*model.QeyKryWEwcUsersVoted, error) {
	result, err := q.DO.Find()
	return result.([]*model.QeyKryWEwcUsersVoted), err
}

func (q qeyKryWEwcUsersVotedDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.QeyKryWEwcUsersVoted, err error) {
	buf := make([]*model.QeyKryWEwcUsersVoted, 0, batchSize)
	err = q.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (q qeyKryWEwcUsersVotedDo) FindInBatches(result *[]*model.QeyKryWEwcUsersVoted, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return q.DO.FindInBatches(result, batchSize, fc)
}

func (q qeyKryWEwcUsersVotedDo) Attrs(attrs ...field.AssignExpr) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Attrs(attrs...))
}

func (q qeyKryWEwcUsersVotedDo) Assign(attrs ...field.AssignExpr) IQeyKryWEwcUsersVotedDo {
	return q.withDO(q.DO.Assign(attrs...))
}

func (q qeyKryWEwcUsersVotedDo) Joins(fields ...field.RelationField) IQeyKryWEwcUsersVotedDo {
	for _, _f := range fields {
		q = *q.withDO(q.DO.Joins(_f))
	}
	return &q
}

func (q qeyKryWEwcUsersVotedDo) Preload(fields ...field.RelationField) IQeyKryWEwcUsersVotedDo {
	for _, _f := range fields {
		q = *q.withDO(q.DO.Preload(_f))
	}
	return &q
}

func (q qeyKryWEwcUsersVotedDo) FirstOrInit() (*model.QeyKryWEwcUsersVoted, error) {
	if result, err := q.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwcUsersVoted), nil
	}
}

func (q qeyKryWEwcUsersVotedDo) FirstOrCreate() (*model.QeyKryWEwcUsersVoted, error) {
	if result, err := q.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwcUsersVoted), nil
	}
}

func (q qeyKryWEwcUsersVotedDo) FindByPage(offset int, limit int) (result []*model.QeyKryWEwcUsersVoted, count int64, err error) {
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

func (q qeyKryWEwcUsersVotedDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = q.Count()
	if err != nil {
		return
	}

	err = q.Offset(offset).Limit(limit).Scan(result)
	return
}

func (q qeyKryWEwcUsersVotedDo) Scan(result interface{}) (err error) {
	return q.DO.Scan(result)
}

func (q qeyKryWEwcUsersVotedDo) Delete(models ...*model.QeyKryWEwcUsersVoted) (result gen.ResultInfo, err error) {
	return q.DO.Delete(models)
}

func (q *qeyKryWEwcUsersVotedDo) withDO(do gen.Dao) *qeyKryWEwcUsersVotedDo {
	q.DO = *do.(*gen.DO)
	return q
}
