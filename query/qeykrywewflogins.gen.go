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

func newQeyKryWEwflogin(db *gorm.DB, opts ...gen.DOOption) qeyKryWEwflogin {
	_qeyKryWEwflogin := qeyKryWEwflogin{}

	_qeyKryWEwflogin.qeyKryWEwfloginDo.UseDB(db, opts...)
	_qeyKryWEwflogin.qeyKryWEwfloginDo.UseModel(&model.QeyKryWEwflogin{})

	tableName := _qeyKryWEwflogin.qeyKryWEwfloginDo.TableName()
	_qeyKryWEwflogin.ALL = field.NewAsterisk(tableName)
	_qeyKryWEwflogin.ID = field.NewInt32(tableName, "id")
	_qeyKryWEwflogin.HitID = field.NewInt32(tableName, "hitID")
	_qeyKryWEwflogin.Ctime = field.NewFloat64(tableName, "ctime")
	_qeyKryWEwflogin.Fail = field.NewInt32(tableName, "fail")
	_qeyKryWEwflogin.Action = field.NewString(tableName, "action")
	_qeyKryWEwflogin.Username = field.NewString(tableName, "username")
	_qeyKryWEwflogin.UserID = field.NewInt32(tableName, "userID")
	_qeyKryWEwflogin.IP = field.NewBytes(tableName, "IP")
	_qeyKryWEwflogin.UA = field.NewString(tableName, "UA")

	_qeyKryWEwflogin.fillFieldMap()

	return _qeyKryWEwflogin
}

type qeyKryWEwflogin struct {
	qeyKryWEwfloginDo

	ALL      field.Asterisk
	ID       field.Int32
	HitID    field.Int32
	Ctime    field.Float64
	Fail     field.Int32
	Action   field.String
	Username field.String
	UserID   field.Int32
	IP       field.Bytes
	UA       field.String

	fieldMap map[string]field.Expr
}

func (q qeyKryWEwflogin) Table(newTableName string) *qeyKryWEwflogin {
	q.qeyKryWEwfloginDo.UseTable(newTableName)
	return q.updateTableName(newTableName)
}

func (q qeyKryWEwflogin) As(alias string) *qeyKryWEwflogin {
	q.qeyKryWEwfloginDo.DO = *(q.qeyKryWEwfloginDo.As(alias).(*gen.DO))
	return q.updateTableName(alias)
}

func (q *qeyKryWEwflogin) updateTableName(table string) *qeyKryWEwflogin {
	q.ALL = field.NewAsterisk(table)
	q.ID = field.NewInt32(table, "id")
	q.HitID = field.NewInt32(table, "hitID")
	q.Ctime = field.NewFloat64(table, "ctime")
	q.Fail = field.NewInt32(table, "fail")
	q.Action = field.NewString(table, "action")
	q.Username = field.NewString(table, "username")
	q.UserID = field.NewInt32(table, "userID")
	q.IP = field.NewBytes(table, "IP")
	q.UA = field.NewString(table, "UA")

	q.fillFieldMap()

	return q
}

func (q *qeyKryWEwflogin) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := q.fieldMap[fieldName]
	if !ok || _f == nil {
		return nil, false
	}
	_oe, ok := _f.(field.OrderExpr)
	return _oe, ok
}

func (q *qeyKryWEwflogin) fillFieldMap() {
	q.fieldMap = make(map[string]field.Expr, 9)
	q.fieldMap["id"] = q.ID
	q.fieldMap["hitID"] = q.HitID
	q.fieldMap["ctime"] = q.Ctime
	q.fieldMap["fail"] = q.Fail
	q.fieldMap["action"] = q.Action
	q.fieldMap["username"] = q.Username
	q.fieldMap["userID"] = q.UserID
	q.fieldMap["IP"] = q.IP
	q.fieldMap["UA"] = q.UA
}

func (q qeyKryWEwflogin) clone(db *gorm.DB) qeyKryWEwflogin {
	q.qeyKryWEwfloginDo.ReplaceConnPool(db.Statement.ConnPool)
	return q
}

func (q qeyKryWEwflogin) replaceDB(db *gorm.DB) qeyKryWEwflogin {
	q.qeyKryWEwfloginDo.ReplaceDB(db)
	return q
}

type qeyKryWEwfloginDo struct{ gen.DO }

type IQeyKryWEwfloginDo interface {
	gen.SubQuery
	Debug() IQeyKryWEwfloginDo
	WithContext(ctx context.Context) IQeyKryWEwfloginDo
	WithResult(fc func(tx gen.Dao)) gen.ResultInfo
	ReplaceDB(db *gorm.DB)
	ReadDB() IQeyKryWEwfloginDo
	WriteDB() IQeyKryWEwfloginDo
	As(alias string) gen.Dao
	Session(config *gorm.Session) IQeyKryWEwfloginDo
	Columns(cols ...field.Expr) gen.Columns
	Clauses(conds ...clause.Expression) IQeyKryWEwfloginDo
	Not(conds ...gen.Condition) IQeyKryWEwfloginDo
	Or(conds ...gen.Condition) IQeyKryWEwfloginDo
	Select(conds ...field.Expr) IQeyKryWEwfloginDo
	Where(conds ...gen.Condition) IQeyKryWEwfloginDo
	Order(conds ...field.Expr) IQeyKryWEwfloginDo
	Distinct(cols ...field.Expr) IQeyKryWEwfloginDo
	Omit(cols ...field.Expr) IQeyKryWEwfloginDo
	Join(table schema.Tabler, on ...field.Expr) IQeyKryWEwfloginDo
	LeftJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEwfloginDo
	RightJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEwfloginDo
	Group(cols ...field.Expr) IQeyKryWEwfloginDo
	Having(conds ...gen.Condition) IQeyKryWEwfloginDo
	Limit(limit int) IQeyKryWEwfloginDo
	Offset(offset int) IQeyKryWEwfloginDo
	Count() (count int64, err error)
	Scopes(funcs ...func(gen.Dao) gen.Dao) IQeyKryWEwfloginDo
	Unscoped() IQeyKryWEwfloginDo
	Create(values ...*model.QeyKryWEwflogin) error
	CreateInBatches(values []*model.QeyKryWEwflogin, batchSize int) error
	Save(values ...*model.QeyKryWEwflogin) error
	First() (*model.QeyKryWEwflogin, error)
	Take() (*model.QeyKryWEwflogin, error)
	Last() (*model.QeyKryWEwflogin, error)
	Find() ([]*model.QeyKryWEwflogin, error)
	FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.QeyKryWEwflogin, err error)
	FindInBatches(result *[]*model.QeyKryWEwflogin, batchSize int, fc func(tx gen.Dao, batch int) error) error
	Pluck(column field.Expr, dest interface{}) error
	Delete(...*model.QeyKryWEwflogin) (info gen.ResultInfo, err error)
	Update(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	Updates(value interface{}) (info gen.ResultInfo, err error)
	UpdateColumn(column field.Expr, value interface{}) (info gen.ResultInfo, err error)
	UpdateColumnSimple(columns ...field.AssignExpr) (info gen.ResultInfo, err error)
	UpdateColumns(value interface{}) (info gen.ResultInfo, err error)
	UpdateFrom(q gen.SubQuery) gen.Dao
	Attrs(attrs ...field.AssignExpr) IQeyKryWEwfloginDo
	Assign(attrs ...field.AssignExpr) IQeyKryWEwfloginDo
	Joins(fields ...field.RelationField) IQeyKryWEwfloginDo
	Preload(fields ...field.RelationField) IQeyKryWEwfloginDo
	FirstOrInit() (*model.QeyKryWEwflogin, error)
	FirstOrCreate() (*model.QeyKryWEwflogin, error)
	FindByPage(offset int, limit int) (result []*model.QeyKryWEwflogin, count int64, err error)
	ScanByPage(result interface{}, offset int, limit int) (count int64, err error)
	Scan(result interface{}) (err error)
	Returning(value interface{}, columns ...string) IQeyKryWEwfloginDo
	UnderlyingDB() *gorm.DB
	schema.Tabler
}

func (q qeyKryWEwfloginDo) Debug() IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Debug())
}

func (q qeyKryWEwfloginDo) WithContext(ctx context.Context) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.WithContext(ctx))
}

func (q qeyKryWEwfloginDo) ReadDB() IQeyKryWEwfloginDo {
	return q.Clauses(dbresolver.Read)
}

func (q qeyKryWEwfloginDo) WriteDB() IQeyKryWEwfloginDo {
	return q.Clauses(dbresolver.Write)
}

func (q qeyKryWEwfloginDo) Session(config *gorm.Session) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Session(config))
}

func (q qeyKryWEwfloginDo) Clauses(conds ...clause.Expression) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Clauses(conds...))
}

func (q qeyKryWEwfloginDo) Returning(value interface{}, columns ...string) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Returning(value, columns...))
}

func (q qeyKryWEwfloginDo) Not(conds ...gen.Condition) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Not(conds...))
}

func (q qeyKryWEwfloginDo) Or(conds ...gen.Condition) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Or(conds...))
}

func (q qeyKryWEwfloginDo) Select(conds ...field.Expr) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Select(conds...))
}

func (q qeyKryWEwfloginDo) Where(conds ...gen.Condition) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Where(conds...))
}

func (q qeyKryWEwfloginDo) Order(conds ...field.Expr) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Order(conds...))
}

func (q qeyKryWEwfloginDo) Distinct(cols ...field.Expr) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Distinct(cols...))
}

func (q qeyKryWEwfloginDo) Omit(cols ...field.Expr) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Omit(cols...))
}

func (q qeyKryWEwfloginDo) Join(table schema.Tabler, on ...field.Expr) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Join(table, on...))
}

func (q qeyKryWEwfloginDo) LeftJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.LeftJoin(table, on...))
}

func (q qeyKryWEwfloginDo) RightJoin(table schema.Tabler, on ...field.Expr) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.RightJoin(table, on...))
}

func (q qeyKryWEwfloginDo) Group(cols ...field.Expr) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Group(cols...))
}

func (q qeyKryWEwfloginDo) Having(conds ...gen.Condition) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Having(conds...))
}

func (q qeyKryWEwfloginDo) Limit(limit int) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Limit(limit))
}

func (q qeyKryWEwfloginDo) Offset(offset int) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Offset(offset))
}

func (q qeyKryWEwfloginDo) Scopes(funcs ...func(gen.Dao) gen.Dao) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Scopes(funcs...))
}

func (q qeyKryWEwfloginDo) Unscoped() IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Unscoped())
}

func (q qeyKryWEwfloginDo) Create(values ...*model.QeyKryWEwflogin) error {
	if len(values) == 0 {
		return nil
	}
	return q.DO.Create(values)
}

func (q qeyKryWEwfloginDo) CreateInBatches(values []*model.QeyKryWEwflogin, batchSize int) error {
	return q.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (q qeyKryWEwfloginDo) Save(values ...*model.QeyKryWEwflogin) error {
	if len(values) == 0 {
		return nil
	}
	return q.DO.Save(values)
}

func (q qeyKryWEwfloginDo) First() (*model.QeyKryWEwflogin, error) {
	if result, err := q.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwflogin), nil
	}
}

func (q qeyKryWEwfloginDo) Take() (*model.QeyKryWEwflogin, error) {
	if result, err := q.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwflogin), nil
	}
}

func (q qeyKryWEwfloginDo) Last() (*model.QeyKryWEwflogin, error) {
	if result, err := q.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwflogin), nil
	}
}

func (q qeyKryWEwfloginDo) Find() ([]*model.QeyKryWEwflogin, error) {
	result, err := q.DO.Find()
	return result.([]*model.QeyKryWEwflogin), err
}

func (q qeyKryWEwfloginDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*model.QeyKryWEwflogin, err error) {
	buf := make([]*model.QeyKryWEwflogin, 0, batchSize)
	err = q.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (q qeyKryWEwfloginDo) FindInBatches(result *[]*model.QeyKryWEwflogin, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return q.DO.FindInBatches(result, batchSize, fc)
}

func (q qeyKryWEwfloginDo) Attrs(attrs ...field.AssignExpr) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Attrs(attrs...))
}

func (q qeyKryWEwfloginDo) Assign(attrs ...field.AssignExpr) IQeyKryWEwfloginDo {
	return q.withDO(q.DO.Assign(attrs...))
}

func (q qeyKryWEwfloginDo) Joins(fields ...field.RelationField) IQeyKryWEwfloginDo {
	for _, _f := range fields {
		q = *q.withDO(q.DO.Joins(_f))
	}
	return &q
}

func (q qeyKryWEwfloginDo) Preload(fields ...field.RelationField) IQeyKryWEwfloginDo {
	for _, _f := range fields {
		q = *q.withDO(q.DO.Preload(_f))
	}
	return &q
}

func (q qeyKryWEwfloginDo) FirstOrInit() (*model.QeyKryWEwflogin, error) {
	if result, err := q.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwflogin), nil
	}
}

func (q qeyKryWEwfloginDo) FirstOrCreate() (*model.QeyKryWEwflogin, error) {
	if result, err := q.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*model.QeyKryWEwflogin), nil
	}
}

func (q qeyKryWEwfloginDo) FindByPage(offset int, limit int) (result []*model.QeyKryWEwflogin, count int64, err error) {
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

func (q qeyKryWEwfloginDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = q.Count()
	if err != nil {
		return
	}

	err = q.Offset(offset).Limit(limit).Scan(result)
	return
}

func (q qeyKryWEwfloginDo) Scan(result interface{}) (err error) {
	return q.DO.Scan(result)
}

func (q qeyKryWEwfloginDo) Delete(models ...*model.QeyKryWEwflogin) (result gen.ResultInfo, err error) {
	return q.DO.Delete(models)
}

func (q *qeyKryWEwfloginDo) withDO(do gen.Dao) *qeyKryWEwfloginDo {
	q.DO = *do.(*gen.DO)
	return q
}
