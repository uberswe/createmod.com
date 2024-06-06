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

func newQeyKryWEwpmailsmtpTasksMetum(db *gorm.DB, opts ...gen.DOOption) qeyKryWEwpmailsmtpTasksMetum {
	_qeyKryWEwpmailsmtpTasksMetum := qeyKryWEwpmailsmtpTasksMetum{}

	_qeyKryWEwpmailsmtpTasksMetum.qeyKryWEwpmailsmtpTasksMetumDo.UseDB(db, opts...)
	_qeyKryWEwpmailsmtpTasksMetum.qeyKryWEwpmailsmtpTasksMetumDo.UseModel(&model.QeyKryWEwpmailsmtpTasksMetum{})

	tableName := _qeyKryWEwpmailsmtpTasksMetum.qeyKryWEwpmailsmtpTasksMetumDo.TableName()
	_qeyKryWEwpmailsmtpTasksMetum.ALL = field.NewAsterisk(tableName)
	_qeyKryWEwpmailsmtpTasksMetum.ID = field.NewInt64(tableName, "id")
	_qeyKryWEwpmailsmtpTasksMetum.Action = field.NewString(tableName, "action")
	_qeyKryWEwpmailsmtpTasksMetum.Data = field.NewString(tableName, "data")
	_qeyKryWEwpmailsmtpTasksMetum.Date = field.NewTime(tableName, "date")

	_qeyKryWEwpmailsmtpTasksMetum.fillFieldMap()

	return _qeyKryWEwpmailsmtpTasksMetum
}

type qeyKryWEwpmailsmtpTasksMetum struct {
	qeyKryWEwpmailsmtpTasksMetumDo

	ALL    field.Asterisk
	ID     field.Int64
	Action field.String
	Data   field.String
	Date   field.Time

	fieldMap map[string]field.Expr
}

func (q qeyKryWEwpmailsmtpTasksMetum) Table(newTableName string) *qeyKryWEwpmailsmtpTasksMetum {
	q.qeyKryWEwpmailsmtpTasksMetumDo.UseTable(newTableName)
	return q.updateTableName(newTableName)
}

func (q qeyKryWEwpmailsmtpTasksMetum) As(alias string) *qeyKryWEwpmailsmtpTasksMetum {
	q.qeyKryWEwpmailsmtpTasksMetumDo.DO = *(q.qeyKryWEwpmailsmtpTasksMetumDo.As(alias).(*gen.DO))
	return q.updateTableName(alias)
}

func (q *qeyKryWEwpmailsmtpTasksMetum) updateTableName(table string) *qeyKryWEwpmailsmtpTasksMetum {
	q.ALL = field.NewAsterisk(table)
	q.ID = field.NewInt64(table, "id")
	q.Action = field.NewString(table, "action")
	q.Data = field.NewString(table, "data")
	q.Date = field.NewTime(table, "date")

	q.fillFieldMap()

	return q
}

func (q *qeyKryWEwpmailsmtpTasksMetum) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := q.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (q *qeyKryWEwpmailsmtpTasksMetum) fillFieldMap() {
	q.fieldMap = make(map[string]field.Expr, 4)
	q.fieldMap["id"] = q.ID
	q.fieldMap["action"] = q.Action
	q.fieldMap["data"] = q.Data
	q.fieldMap["date"] = q.Date
}

func (q qeyKryWEwpmailsmtpTasksMetum) clone(db *gorm.DB) qeyKryWEwpmailsmtpTasksMetum {
	q.qeyKryWEwpmailsmtpTasksMetumDo.ReplaceConnPool(db.Statement.ConnPool)
	return q
}

func (q qeyKryWEwpmailsmtpTasksMetum) replaceDB(db *gorm.DB) qeyKryWEwpmailsmtpTasksMetum {
	q.qeyKryWEwpmailsmtpTasksMetumDo.ReplaceDB(db)
	return q
}

type qeyKryWEwpmailsmtpTasksMetumDo struct{ gen.DO }

type IQeyKryWEwpmailsmtpTasksMetumDo interface {
	gen.SubQuery
	Debug() IQeyKryWEwpmailsmtpTasksMetumDo
	WithContext(ctx context.Context) IQeyKryWEwpmailsmtpTasksMetumDo
	WithResult(fc func(tx gen.Dao)) gen.ResultInfo
	ReplaceDB(db *gorm.DB)
	ReadDB() IQeyKryWEwpmailsmtpTasksMetumDo
	WriteDB() IQeyKryWEwpmailsmtpTasksMetumDo
	As(alias string) gen.Dao
	Session(config *gorm.Session) IQeyKryWEwpmailsmtpTasksMetumDo
	Columns(cols ...field.Expr) gen.Columns
	Clauses(conds ...clause.Expression) IQeyKryWEwpmailsmtpTasksMetumDo
	Not(conds ...gen.Condition) IQeyKryWEwpmailsmtpTasksMetumDo
	Or(conds ...gen.Condition) IQeyKryWEwpmailsmtpTasksMetumDo
	Select(conds ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo
	Where(conds ...gen.Condition) IQeyKryWEwpmailsmtpTasksMetumDo
	Order(conds ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo
	Distinct(cols ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo
	Omit(cols ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo
	Join(table schema.Tabler, on ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo
	LeftJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo
	RightJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo
	Group(cols ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo
	Having(conds ...gen.Condition) IQeyKryWEwpmailsmtpTasksMetumDo
	Limit(limit int) IQeyKryWEwpmailsmtpTasksMetumDo
	Offset(offset int) IQeyKryWEwpmailsmtpTasksMetumDo
	Count() (count int64, err error)
	Scopes(funcs ...func(gen.Dao) gen.Dao) IQeyKryWEwpmailsmtpTasksMetumDo
	Unscoped() IQeyKryWEwpmailsmtpTasksMetumDo
	Create(values ...*model.QeyKryWEwpmailsmtpTasksMetum) error
	CreateInBatches(values []*model.QeyKryWEwpmailsmtpTasksMetum, batchSize int) error
	Save(values ...*model.QeyKryWEwpmailsmtpTasksMetum) error
	First() (*model.QeyKryWEwpmailsmtpTasksMetum, error)
	Take() (*model.QeyKryWEwpmailsmtpTasksMetum, error)
	Last() (*model.QeyKryWEwpmailsmtpTasksMetum, error)
	Find() ([]*model.QeyKryWEwpmailsmtpTasksMetum, error)
	FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.QeyKryWEwpmailsmtpTasksMetum, err error)
	FindInBatches(result *[]*model.QeyKryWEwpmailsmtpTasksMetum, batchSize int, fc func(tx gen.Dao, batch int) error) error
	Pluck(column field.Expr, dest interface{}) error
	Delete(...*model.QeyKryWEwpmailsmtpTasksMetum) (info gen.ResultInfo, err error)
	Update(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	Updates(value interface{}) (info gen.ResultInfo, err error)
	UpdateColumn(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateColumnSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	UpdateColumns(value interface{}) (info gen.ResultInfo, err error)
	UpdateFrom(q gen.SubQuery) gen.Dao
	Attrs(attrs ...field.AssignExpr) IQeyKryWEwpmailsmtpTasksMetumDo
	Assign(attrs ...field.AssignExpr) IQeyKryWEwpmailsmtpTasksMetumDo
	Joins(fields ...field.RelationField) IQeyKryWEwpmailsmtpTasksMetumDo
	Preload(fields ...field.RelationField) IQeyKryWEwpmailsmtpTasksMetumDo
	FirstOrInit() (*model.QeyKryWEwpmailsmtpTasksMetum, error)
	FirstOrCreate() (*model.QeyKryWEwpmailsmtpTasksMetum, error)
	FindByPage(offset int, limit int) (result []*model.QeyKryWEwpmailsmtpTasksMetum, count int64, err error)
	ScanByPage(result interface{}, offset int, limit int) (count int64, err error)
	Scan(result interface{}) (err error)
	Returning(value interface{}, columns ...string) IQeyKryWEwpmailsmtpTasksMetumDo
	UnderlyingDB() *gorm.DB
	schema.Tabler
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Debug() IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Debug())
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) WithContext(ctx context.Context) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.WithContext(ctx))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) ReadDB() IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.Clauses(dbresolver.Read)
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) WriteDB() IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.Clauses(dbresolver.Write)
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Session(config *gorm.Session) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Session(config))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Clauses(conds ...clause.Expression) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Clauses(conds...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Returning(value interface{}, columns ...string) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Returning(value, columns...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Not(conds ...gen.Condition) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Not(conds...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Or(conds ...gen.Condition) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Or(conds...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Select(conds ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Select(conds...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Where(conds ...gen.Condition) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Where(conds...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Order(conds ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Order(conds...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Distinct(cols ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Distinct(cols...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Omit(cols ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Omit(cols...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Join(table schema.Tabler, on ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Join(table, on...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) LeftJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.LeftJoin(table, on...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) RightJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.RightJoin(table, on...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Group(cols ...field.Expr) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Group(cols...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Having(conds ...gen.Condition) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Having(conds...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Limit(limit int) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Limit(limit))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Offset(offset int) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Offset(offset))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Scopes(funcs ...func(gen.Dao) gen.Dao) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Scopes(funcs...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Unscoped() IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Unscoped())
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Create(values ...*model.QeyKryWEwpmailsmtpTasksMetum) error {
	if len(values) == 0 {
		return nil
	}
	return q.DO.Create(values)
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) CreateInBatches(values []*model.QeyKryWEwpmailsmtpTasksMetum, batchSize int) error {
	return q.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (q qeyKryWEwpmailsmtpTasksMetumDo) Save(values ...*model.QeyKryWEwpmailsmtpTasksMetum) error {
	if len(values) == 0 {
		return nil
	}
	return q.DO.Save(values)
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) First() (*model.QeyKryWEwpmailsmtpTasksMetum, error) {
	if result, err := q.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwpmailsmtpTasksMetum), nil
	}
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Take() (*model.QeyKryWEwpmailsmtpTasksMetum, error) {
	if result, err := q.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwpmailsmtpTasksMetum), nil
	}
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Last() (*model.QeyKryWEwpmailsmtpTasksMetum, error) {
	if result, err := q.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwpmailsmtpTasksMetum), nil
	}
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Find() ([]*model.QeyKryWEwpmailsmtpTasksMetum, error) {
	result, err := q.DO.Find()
	return result.([]*model.QeyKryWEwpmailsmtpTasksMetum), err
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.QeyKryWEwpmailsmtpTasksMetum, err error) {
	buf := make([]*model.QeyKryWEwpmailsmtpTasksMetum, 0, batchSize)
	err = q.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) FindInBatches(result *[]*model.QeyKryWEwpmailsmtpTasksMetum, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return q.DO.FindInBatches(result, batchSize, fc)
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Attrs(attrs ...field.AssignExpr) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Attrs(attrs...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Assign(attrs ...field.AssignExpr) IQeyKryWEwpmailsmtpTasksMetumDo {
	return q.withDO(q.DO.Assign(attrs...))
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Joins(fields ...field.RelationField) IQeyKryWEwpmailsmtpTasksMetumDo {
	for _, _f := range fields {
		q = *q.withDO(q.DO.Joins(_f))
	}
	return &q
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Preload(fields ...field.RelationField) IQeyKryWEwpmailsmtpTasksMetumDo {
	for _, _f := range fields {
		q = *q.withDO(q.DO.Preload(_f))
	}
	return &q
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) FirstOrInit() (*model.QeyKryWEwpmailsmtpTasksMetum, error) {
	if result, err := q.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwpmailsmtpTasksMetum), nil
	}
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) FirstOrCreate() (*model.QeyKryWEwpmailsmtpTasksMetum, error) {
	if result, err := q.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwpmailsmtpTasksMetum), nil
	}
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) FindByPage(offset int, limit int) (result []*model.QeyKryWEwpmailsmtpTasksMetum, count int64, err error) {
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

func (q qeyKryWEwpmailsmtpTasksMetumDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = q.Count()
	if err != nil {
		return
	}

	err = q.Offset(offset).Limit(limit).Scan(result)
	return
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Scan(result interface{}) (err error) {
	return q.DO.Scan(result)
}

func (q qeyKryWEwpmailsmtpTasksMetumDo) Delete(models ...*model.QeyKryWEwpmailsmtpTasksMetum) (result gen.ResultInfo, err error) {
	return q.DO.Delete(models)
}

func (q *qeyKryWEwpmailsmtpTasksMetumDo) withDO(do gen.Dao) *qeyKryWEwpmailsmtpTasksMetumDo {
	q.DO = *do.(*gen.DO)
	return q
}
