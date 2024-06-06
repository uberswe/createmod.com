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

func newQeyKryWEusermetum(db *gorm.DB, opts ...gen.DOOption) qeyKryWEusermetum {
	_qeyKryWEusermetum := qeyKryWEusermetum{}

	_qeyKryWEusermetum.qeyKryWEusermetumDo.UseDB(db, opts...)
	_qeyKryWEusermetum.qeyKryWEusermetumDo.UseModel(&model.QeyKryWEusermetum{})

	tableName := _qeyKryWEusermetum.qeyKryWEusermetumDo.TableName()
	_qeyKryWEusermetum.ALL = field.NewAsterisk(tableName)
	_qeyKryWEusermetum.UmetaID = field.NewInt64(tableName, "umeta_id")
	_qeyKryWEusermetum.UserID = field.NewInt64(tableName, "user_id")
	_qeyKryWEusermetum.MetaKey = field.NewString(tableName, "meta_key")
	_qeyKryWEusermetum.MetaValue = field.NewString(tableName, "meta_value")

	_qeyKryWEusermetum.fillFieldMap()

	return _qeyKryWEusermetum
}

type qeyKryWEusermetum struct {
	qeyKryWEusermetumDo

	ALL       field.Asterisk
	UmetaID   field.Int64
	UserID    field.Int64
	MetaKey   field.String
	MetaValue field.String

	fieldMap map[string]field.Expr
}

func (q qeyKryWEusermetum) Table(newTableName string) *qeyKryWEusermetum {
	q.qeyKryWEusermetumDo.UseTable(newTableName)
	return q.updateTableName(newTableName)
}

func (q qeyKryWEusermetum) As(alias string) *qeyKryWEusermetum {
	q.qeyKryWEusermetumDo.DO = *(q.qeyKryWEusermetumDo.As(alias).(*gen.DO))
	return q.updateTableName(alias)
}

func (q *qeyKryWEusermetum) updateTableName(table string) *qeyKryWEusermetum {
	q.ALL = field.NewAsterisk(table)
	q.UmetaID = field.NewInt64(table, "umeta_id")
	q.UserID = field.NewInt64(table, "user_id")
	q.MetaKey = field.NewString(table, "meta_key")
	q.MetaValue = field.NewString(table, "meta_value")

	q.fillFieldMap()

	return q
}

func (q *qeyKryWEusermetum) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := q.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (q *qeyKryWEusermetum) fillFieldMap() {
	q.fieldMap = make(map[string]field.Expr, 4)
	q.fieldMap["umeta_id"] = q.UmetaID
	q.fieldMap["user_id"] = q.UserID
	q.fieldMap["meta_key"] = q.MetaKey
	q.fieldMap["meta_value"] = q.MetaValue
}

func (q qeyKryWEusermetum) clone(db *gorm.DB) qeyKryWEusermetum {
	q.qeyKryWEusermetumDo.ReplaceConnPool(db.Statement.ConnPool)
	return q
}

func (q qeyKryWEusermetum) replaceDB(db *gorm.DB) qeyKryWEusermetum {
	q.qeyKryWEusermetumDo.ReplaceDB(db)
	return q
}

type qeyKryWEusermetumDo struct{ gen.DO }

type IQeyKryWEusermetumDo interface {
	gen.SubQuery
	Debug() IQeyKryWEusermetumDo
	WithContext(ctx context.Context) IQeyKryWEusermetumDo
	WithResult(fc func(tx gen.Dao)) gen.ResultInfo
	ReplaceDB(db *gorm.DB)
	ReadDB() IQeyKryWEusermetumDo
	WriteDB() IQeyKryWEusermetumDo
	As(alias string) gen.Dao
	Session(config *gorm.Session) IQeyKryWEusermetumDo
	Columns(cols ...field.Expr) gen.Columns
	Clauses(conds ...clause.Expression) IQeyKryWEusermetumDo
	Not(conds ...gen.Condition) IQeyKryWEusermetumDo
	Or(conds ...gen.Condition) IQeyKryWEusermetumDo
	Select(conds ...field.Expr) IQeyKryWEusermetumDo
	Where(conds ...gen.Condition) IQeyKryWEusermetumDo
	Order(conds ...field.Expr) IQeyKryWEusermetumDo
	Distinct(cols ...field.Expr) IQeyKryWEusermetumDo
	Omit(cols ...field.Expr) IQeyKryWEusermetumDo
	Join(table schema.Tabler, on ...field.Expr) IQeyKryWEusermetumDo
	LeftJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEusermetumDo
	RightJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEusermetumDo
	Group(cols ...field.Expr) IQeyKryWEusermetumDo
	Having(conds ...gen.Condition) IQeyKryWEusermetumDo
	Limit(limit int) IQeyKryWEusermetumDo
	Offset(offset int) IQeyKryWEusermetumDo
	Count() (count int64, err error)
	Scopes(funcs ...func(gen.Dao) gen.Dao) IQeyKryWEusermetumDo
	Unscoped() IQeyKryWEusermetumDo
	Create(values ...*model.QeyKryWEusermetum) error
	CreateInBatches(values []*model.QeyKryWEusermetum, batchSize int) error
	Save(values ...*model.QeyKryWEusermetum) error
	First() (*model.QeyKryWEusermetum, error)
	Take() (*model.QeyKryWEusermetum, error)
	Last() (*model.QeyKryWEusermetum, error)
	Find() ([]*model.QeyKryWEusermetum, error)
	FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.QeyKryWEusermetum, err error)
	FindInBatches(result *[]*model.QeyKryWEusermetum, batchSize int, fc func(tx gen.Dao, batch int) error) error
	Pluck(column field.Expr, dest interface{}) error
	Delete(...*model.QeyKryWEusermetum) (info gen.ResultInfo, err error)
	Update(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	Updates(value interface{}) (info gen.ResultInfo, err error)
	UpdateColumn(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateColumnSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	UpdateColumns(value interface{}) (info gen.ResultInfo, err error)
	UpdateFrom(q gen.SubQuery) gen.Dao
	Attrs(attrs ...field.AssignExpr) IQeyKryWEusermetumDo
	Assign(attrs ...field.AssignExpr) IQeyKryWEusermetumDo
	Joins(fields ...field.RelationField) IQeyKryWEusermetumDo
	Preload(fields ...field.RelationField) IQeyKryWEusermetumDo
	FirstOrInit() (*model.QeyKryWEusermetum, error)
	FirstOrCreate() (*model.QeyKryWEusermetum, error)
	FindByPage(offset int, limit int) (result []*model.QeyKryWEusermetum, count int64, err error)
	ScanByPage(result interface{}, offset int, limit int) (count int64, err error)
	Scan(result interface{}) (err error)
	Returning(value interface{}, columns ...string) IQeyKryWEusermetumDo
	UnderlyingDB() *gorm.DB
	schema.Tabler
}

func (q qeyKryWEusermetumDo) Debug() IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Debug())
}

func (q qeyKryWEusermetumDo) WithContext(ctx context.Context) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.WithContext(ctx))
}

func (q qeyKryWEusermetumDo) ReadDB() IQeyKryWEusermetumDo {
	return q.Clauses(dbresolver.Read)
}

func (q qeyKryWEusermetumDo) WriteDB() IQeyKryWEusermetumDo {
	return q.Clauses(dbresolver.Write)
}

func (q qeyKryWEusermetumDo) Session(config *gorm.Session) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Session(config))
}

func (q qeyKryWEusermetumDo) Clauses(conds ...clause.Expression) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Clauses(conds...))
}

func (q qeyKryWEusermetumDo) Returning(value interface{}, columns ...string) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Returning(value, columns...))
}

func (q qeyKryWEusermetumDo) Not(conds ...gen.Condition) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Not(conds...))
}

func (q qeyKryWEusermetumDo) Or(conds ...gen.Condition) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Or(conds...))
}

func (q qeyKryWEusermetumDo) Select(conds ...field.Expr) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Select(conds...))
}

func (q qeyKryWEusermetumDo) Where(conds ...gen.Condition) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Where(conds...))
}

func (q qeyKryWEusermetumDo) Order(conds ...field.Expr) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Order(conds...))
}

func (q qeyKryWEusermetumDo) Distinct(cols ...field.Expr) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Distinct(cols...))
}

func (q qeyKryWEusermetumDo) Omit(cols ...field.Expr) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Omit(cols...))
}

func (q qeyKryWEusermetumDo) Join(table schema.Tabler, on ...field.Expr) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Join(table, on...))
}

func (q qeyKryWEusermetumDo) LeftJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.LeftJoin(table, on...))
}

func (q qeyKryWEusermetumDo) RightJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.RightJoin(table, on...))
}

func (q qeyKryWEusermetumDo) Group(cols ...field.Expr) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Group(cols...))
}

func (q qeyKryWEusermetumDo) Having(conds ...gen.Condition) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Having(conds...))
}

func (q qeyKryWEusermetumDo) Limit(limit int) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Limit(limit))
}

func (q qeyKryWEusermetumDo) Offset(offset int) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Offset(offset))
}

func (q qeyKryWEusermetumDo) Scopes(funcs ...func(gen.Dao) gen.Dao) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Scopes(funcs...))
}

func (q qeyKryWEusermetumDo) Unscoped() IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Unscoped())
}

func (q qeyKryWEusermetumDo) Create(values ...*model.QeyKryWEusermetum) error {
	if len(values) == 0 {
		return nil
	}
	return q.DO.Create(values)
}

func (q qeyKryWEusermetumDo) CreateInBatches(values []*model.QeyKryWEusermetum, batchSize int) error {
	return q.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (q qeyKryWEusermetumDo) Save(values ...*model.QeyKryWEusermetum) error {
	if len(values) == 0 {
		return nil
	}
	return q.DO.Save(values)
}

func (q qeyKryWEusermetumDo) First() (*model.QeyKryWEusermetum, error) {
	if result, err := q.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEusermetum), nil
	}
}

func (q qeyKryWEusermetumDo) Take() (*model.QeyKryWEusermetum, error) {
	if result, err := q.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEusermetum), nil
	}
}

func (q qeyKryWEusermetumDo) Last() (*model.QeyKryWEusermetum, error) {
	if result, err := q.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEusermetum), nil
	}
}

func (q qeyKryWEusermetumDo) Find() ([]*model.QeyKryWEusermetum, error) {
	result, err := q.DO.Find()
	return result.([]*model.QeyKryWEusermetum), err
}

func (q qeyKryWEusermetumDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.QeyKryWEusermetum, err error) {
	buf := make([]*model.QeyKryWEusermetum, 0, batchSize)
	err = q.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (q qeyKryWEusermetumDo) FindInBatches(result *[]*model.QeyKryWEusermetum, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return q.DO.FindInBatches(result, batchSize, fc)
}

func (q qeyKryWEusermetumDo) Attrs(attrs ...field.AssignExpr) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Attrs(attrs...))
}

func (q qeyKryWEusermetumDo) Assign(attrs ...field.AssignExpr) IQeyKryWEusermetumDo {
	return q.withDO(q.DO.Assign(attrs...))
}

func (q qeyKryWEusermetumDo) Joins(fields ...field.RelationField) IQeyKryWEusermetumDo {
	for _, _f := range fields {
		q = *q.withDO(q.DO.Joins(_f))
	}
	return &q
}

func (q qeyKryWEusermetumDo) Preload(fields ...field.RelationField) IQeyKryWEusermetumDo {
	for _, _f := range fields {
		q = *q.withDO(q.DO.Preload(_f))
	}
	return &q
}

func (q qeyKryWEusermetumDo) FirstOrInit() (*model.QeyKryWEusermetum, error) {
	if result, err := q.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEusermetum), nil
	}
}

func (q qeyKryWEusermetumDo) FirstOrCreate() (*model.QeyKryWEusermetum, error) {
	if result, err := q.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEusermetum), nil
	}
}

func (q qeyKryWEusermetumDo) FindByPage(offset int, limit int) (result []*model.QeyKryWEusermetum, count int64, err error) {
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

func (q qeyKryWEusermetumDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = q.Count()
	if err != nil {
		return
	}

	err = q.Offset(offset).Limit(limit).Scan(result)
	return
}

func (q qeyKryWEusermetumDo) Scan(result interface{}) (err error) {
	return q.DO.Scan(result)
}

func (q qeyKryWEusermetumDo) Delete(models ...*model.QeyKryWEusermetum) (result gen.ResultInfo, err error) {
	return q.DO.Delete(models)
}

func (q *qeyKryWEusermetumDo) withDO(do gen.Dao) *qeyKryWEusermetumDo {
	q.DO = *do.(*gen.DO)
	return q
}
