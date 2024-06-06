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

func newQeyKryWEsnippet(db *gorm.DB, opts ...gen.DOOption) qeyKryWEsnippet {
	_qeyKryWEsnippet := qeyKryWEsnippet{}

	_qeyKryWEsnippet.qeyKryWEsnippetDo.UseDB(db, opts...)
	_qeyKryWEsnippet.qeyKryWEsnippetDo.UseModel(&model.QeyKryWEsnippet{})

	tableName := _qeyKryWEsnippet.qeyKryWEsnippetDo.TableName()
	_qeyKryWEsnippet.ALL = field.NewAsterisk(tableName)
	_qeyKryWEsnippet.ID = field.NewInt64(tableName, "id")
	_qeyKryWEsnippet.Name = field.NewString(tableName, "name")
	_qeyKryWEsnippet.Description = field.NewString(tableName, "description")
	_qeyKryWEsnippet.Code = field.NewString(tableName, "code")
	_qeyKryWEsnippet.Tags = field.NewString(tableName, "tags")
	_qeyKryWEsnippet.Scope = field.NewString(tableName, "scope")
	_qeyKryWEsnippet.Priority = field.NewInt32(tableName, "priority")
	_qeyKryWEsnippet.Active = field.NewBool(tableName, "active")
	_qeyKryWEsnippet.Modified = field.NewTime(tableName, "modified")

	_qeyKryWEsnippet.fillFieldMap()

	return _qeyKryWEsnippet
}

type qeyKryWEsnippet struct {
	qeyKryWEsnippetDo

	ALL         field.Asterisk
	ID          field.Int64
	Name        field.String
	Description field.String
	Code        field.String
	Tags        field.String
	Scope       field.String
	Priority    field.Int32
	Active      field.Bool
	Modified    field.Time

	fieldMap map[string]field.Expr
}

func (q qeyKryWEsnippet) Table(newTableName string) *qeyKryWEsnippet {
	q.qeyKryWEsnippetDo.UseTable(newTableName)
	return q.updateTableName(newTableName)
}

func (q qeyKryWEsnippet) As(alias string) *qeyKryWEsnippet {
	q.qeyKryWEsnippetDo.DO = *(q.qeyKryWEsnippetDo.As(alias).(*gen.DO))
	return q.updateTableName(alias)
}

func (q *qeyKryWEsnippet) updateTableName(table string) *qeyKryWEsnippet {
	q.ALL = field.NewAsterisk(table)
	q.ID = field.NewInt64(table, "id")
	q.Name = field.NewString(table, "name")
	q.Description = field.NewString(table, "description")
	q.Code = field.NewString(table, "code")
	q.Tags = field.NewString(table, "tags")
	q.Scope = field.NewString(table, "scope")
	q.Priority = field.NewInt32(table, "priority")
	q.Active = field.NewBool(table, "active")
	q.Modified = field.NewTime(table, "modified")

	q.fillFieldMap()

	return q
}

func (q *qeyKryWEsnippet) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := q.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (q *qeyKryWEsnippet) fillFieldMap() {
	q.fieldMap = make(map[string]field.Expr, 9)
	q.fieldMap["id"] = q.ID
	q.fieldMap["name"] = q.Name
	q.fieldMap["description"] = q.Description
	q.fieldMap["code"] = q.Code
	q.fieldMap["tags"] = q.Tags
	q.fieldMap["scope"] = q.Scope
	q.fieldMap["priority"] = q.Priority
	q.fieldMap["active"] = q.Active
	q.fieldMap["modified"] = q.Modified
}

func (q qeyKryWEsnippet) clone(db *gorm.DB) qeyKryWEsnippet {
	q.qeyKryWEsnippetDo.ReplaceConnPool(db.Statement.ConnPool)
	return q
}

func (q qeyKryWEsnippet) replaceDB(db *gorm.DB) qeyKryWEsnippet {
	q.qeyKryWEsnippetDo.ReplaceDB(db)
	return q
}

type qeyKryWEsnippetDo struct{ gen.DO }

type IQeyKryWEsnippetDo interface {
	gen.SubQuery
	Debug() IQeyKryWEsnippetDo
	WithContext(ctx context.Context) IQeyKryWEsnippetDo
	WithResult(fc func(tx gen.Dao)) gen.ResultInfo
	ReplaceDB(db *gorm.DB)
	ReadDB() IQeyKryWEsnippetDo
	WriteDB() IQeyKryWEsnippetDo
	As(alias string) gen.Dao
	Session(config *gorm.Session) IQeyKryWEsnippetDo
	Columns(cols ...field.Expr) gen.Columns
	Clauses(conds ...clause.Expression) IQeyKryWEsnippetDo
	Not(conds ...gen.Condition) IQeyKryWEsnippetDo
	Or(conds ...gen.Condition) IQeyKryWEsnippetDo
	Select(conds ...field.Expr) IQeyKryWEsnippetDo
	Where(conds ...gen.Condition) IQeyKryWEsnippetDo
	Order(conds ...field.Expr) IQeyKryWEsnippetDo
	Distinct(cols ...field.Expr) IQeyKryWEsnippetDo
	Omit(cols ...field.Expr) IQeyKryWEsnippetDo
	Join(table schema.Tabler, on ...field.Expr) IQeyKryWEsnippetDo
	LeftJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEsnippetDo
	RightJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEsnippetDo
	Group(cols ...field.Expr) IQeyKryWEsnippetDo
	Having(conds ...gen.Condition) IQeyKryWEsnippetDo
	Limit(limit int) IQeyKryWEsnippetDo
	Offset(offset int) IQeyKryWEsnippetDo
	Count() (count int64, err error)
	Scopes(funcs ...func(gen.Dao) gen.Dao) IQeyKryWEsnippetDo
	Unscoped() IQeyKryWEsnippetDo
	Create(values ...*model.QeyKryWEsnippet) error
	CreateInBatches(values []*model.QeyKryWEsnippet, batchSize int) error
	Save(values ...*model.QeyKryWEsnippet) error
	First() (*model.QeyKryWEsnippet, error)
	Take() (*model.QeyKryWEsnippet, error)
	Last() (*model.QeyKryWEsnippet, error)
	Find() ([]*model.QeyKryWEsnippet, error)
	FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.QeyKryWEsnippet, err error)
	FindInBatches(result *[]*model.QeyKryWEsnippet, batchSize int, fc func(tx gen.Dao, batch int) error) error
	Pluck(column field.Expr, dest interface{}) error
	Delete(...*model.QeyKryWEsnippet) (info gen.ResultInfo, err error)
	Update(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	Updates(value interface{}) (info gen.ResultInfo, err error)
	UpdateColumn(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateColumnSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	UpdateColumns(value interface{}) (info gen.ResultInfo, err error)
	UpdateFrom(q gen.SubQuery) gen.Dao
	Attrs(attrs ...field.AssignExpr) IQeyKryWEsnippetDo
	Assign(attrs ...field.AssignExpr) IQeyKryWEsnippetDo
	Joins(fields ...field.RelationField) IQeyKryWEsnippetDo
	Preload(fields ...field.RelationField) IQeyKryWEsnippetDo
	FirstOrInit() (*model.QeyKryWEsnippet, error)
	FirstOrCreate() (*model.QeyKryWEsnippet, error)
	FindByPage(offset int, limit int) (result []*model.QeyKryWEsnippet, count int64, err error)
	ScanByPage(result interface{}, offset int, limit int) (count int64, err error)
	Scan(result interface{}) (err error)
	Returning(value interface{}, columns ...string) IQeyKryWEsnippetDo
	UnderlyingDB() *gorm.DB
	schema.Tabler
}

func (q qeyKryWEsnippetDo) Debug() IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Debug())
}

func (q qeyKryWEsnippetDo) WithContext(ctx context.Context) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.WithContext(ctx))
}

func (q qeyKryWEsnippetDo) ReadDB() IQeyKryWEsnippetDo {
	return q.Clauses(dbresolver.Read)
}

func (q qeyKryWEsnippetDo) WriteDB() IQeyKryWEsnippetDo {
	return q.Clauses(dbresolver.Write)
}

func (q qeyKryWEsnippetDo) Session(config *gorm.Session) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Session(config))
}

func (q qeyKryWEsnippetDo) Clauses(conds ...clause.Expression) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Clauses(conds...))
}

func (q qeyKryWEsnippetDo) Returning(value interface{}, columns ...string) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Returning(value, columns...))
}

func (q qeyKryWEsnippetDo) Not(conds ...gen.Condition) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Not(conds...))
}

func (q qeyKryWEsnippetDo) Or(conds ...gen.Condition) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Or(conds...))
}

func (q qeyKryWEsnippetDo) Select(conds ...field.Expr) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Select(conds...))
}

func (q qeyKryWEsnippetDo) Where(conds ...gen.Condition) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Where(conds...))
}

func (q qeyKryWEsnippetDo) Order(conds ...field.Expr) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Order(conds...))
}

func (q qeyKryWEsnippetDo) Distinct(cols ...field.Expr) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Distinct(cols...))
}

func (q qeyKryWEsnippetDo) Omit(cols ...field.Expr) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Omit(cols...))
}

func (q qeyKryWEsnippetDo) Join(table schema.Tabler, on ...field.Expr) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Join(table, on...))
}

func (q qeyKryWEsnippetDo) LeftJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.LeftJoin(table, on...))
}

func (q qeyKryWEsnippetDo) RightJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.RightJoin(table, on...))
}

func (q qeyKryWEsnippetDo) Group(cols ...field.Expr) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Group(cols...))
}

func (q qeyKryWEsnippetDo) Having(conds ...gen.Condition) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Having(conds...))
}

func (q qeyKryWEsnippetDo) Limit(limit int) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Limit(limit))
}

func (q qeyKryWEsnippetDo) Offset(offset int) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Offset(offset))
}

func (q qeyKryWEsnippetDo) Scopes(funcs ...func(gen.Dao) gen.Dao) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Scopes(funcs...))
}

func (q qeyKryWEsnippetDo) Unscoped() IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Unscoped())
}

func (q qeyKryWEsnippetDo) Create(values ...*model.QeyKryWEsnippet) error {
	if len(values) == 0 {
		return nil
	}
	return q.DO.Create(values)
}

func (q qeyKryWEsnippetDo) CreateInBatches(values []*model.QeyKryWEsnippet, batchSize int) error {
	return q.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (q qeyKryWEsnippetDo) Save(values ...*model.QeyKryWEsnippet) error {
	if len(values) == 0 {
		return nil
	}
	return q.DO.Save(values)
}

func (q qeyKryWEsnippetDo) First() (*model.QeyKryWEsnippet, error) {
	if result, err := q.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEsnippet), nil
	}
}

func (q qeyKryWEsnippetDo) Take() (*model.QeyKryWEsnippet, error) {
	if result, err := q.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEsnippet), nil
	}
}

func (q qeyKryWEsnippetDo) Last() (*model.QeyKryWEsnippet, error) {
	if result, err := q.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEsnippet), nil
	}
}

func (q qeyKryWEsnippetDo) Find() ([]*model.QeyKryWEsnippet, error) {
	result, err := q.DO.Find()
	return result.([]*model.QeyKryWEsnippet), err
}

func (q qeyKryWEsnippetDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.QeyKryWEsnippet, err error) {
	buf := make([]*model.QeyKryWEsnippet, 0, batchSize)
	err = q.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (q qeyKryWEsnippetDo) FindInBatches(result *[]*model.QeyKryWEsnippet, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return q.DO.FindInBatches(result, batchSize, fc)
}

func (q qeyKryWEsnippetDo) Attrs(attrs ...field.AssignExpr) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Attrs(attrs...))
}

func (q qeyKryWEsnippetDo) Assign(attrs ...field.AssignExpr) IQeyKryWEsnippetDo {
	return q.withDO(q.DO.Assign(attrs...))
}

func (q qeyKryWEsnippetDo) Joins(fields ...field.RelationField) IQeyKryWEsnippetDo {
	for _, _f := range fields {
		q = *q.withDO(q.DO.Joins(_f))
	}
	return &q
}

func (q qeyKryWEsnippetDo) Preload(fields ...field.RelationField) IQeyKryWEsnippetDo {
	for _, _f := range fields {
		q = *q.withDO(q.DO.Preload(_f))
	}
	return &q
}

func (q qeyKryWEsnippetDo) FirstOrInit() (*model.QeyKryWEsnippet, error) {
	if result, err := q.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEsnippet), nil
	}
}

func (q qeyKryWEsnippetDo) FirstOrCreate() (*model.QeyKryWEsnippet, error) {
	if result, err := q.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEsnippet), nil
	}
}

func (q qeyKryWEsnippetDo) FindByPage(offset int, limit int) (result []*model.QeyKryWEsnippet, count int64, err error) {
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

func (q qeyKryWEsnippetDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = q.Count()
	if err != nil {
		return
	}

	err = q.Offset(offset).Limit(limit).Scan(result)
	return
}

func (q qeyKryWEsnippetDo) Scan(result interface{}) (err error) {
	return q.DO.Scan(result)
}

func (q qeyKryWEsnippetDo) Delete(models ...*model.QeyKryWEsnippet) (result gen.ResultInfo, err error) {
	return q.DO.Delete(models)
}

func (q *qeyKryWEsnippetDo) withDO(do gen.Dao) *qeyKryWEsnippetDo {
	q.DO = *do.(*gen.DO)
	return q
}
