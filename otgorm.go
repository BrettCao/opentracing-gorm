package otgorm

import (
	"context"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
)

type callbackName string

const (
	gormCtx = "gormCtx"

	create   callbackName = "create"
	update   callbackName = "update"
	delete   callbackName = "delete"
	query    callbackName = "query"
	rowQuery callbackName = "row_query"
)

// String returns string.
func (cn callbackName) ToString() string {
	return string(cn)
}

// DB is a wrapper for *gorm.DB.
type DB interface {
	WithContext(ctx context.Context) *gorm.DB
}

type wrapDB struct {
	gorm *gorm.DB
}

// WithContext returns *gorm.DB which is injected a context for tracing.
func (db *wrapDB) WithContext(ctx context.Context) *gorm.DB {
	if ctx == nil {
		return db.gorm
	}

	return db.gorm.Set(gormCtx, ctx)
}

func (db *wrapDB) beforeCreate(scope *gorm.Scope) {
	db.beforeCallback(scope, create)
}

func (db *wrapDB) beforeUpdate(scope *gorm.Scope) {
	db.beforeCallback(scope, update)
}

func (db *wrapDB) beforeDelete(scope *gorm.Scope) {
	db.beforeCallback(scope, delete)
}

func (db *wrapDB) beforeQuery(scope *gorm.Scope) {
	db.beforeCallback(scope, query)
}

func (db *wrapDB) beforeRowQuery(scope *gorm.Scope) {
	db.beforeCallback(scope, rowQuery)
}

// registerCallback is to register callback functions
func (db *wrapDB) registerCallbacks() {
	callbackFmt := "gorm:%s"
	beforeFmt := "ot:before_%s"
	afterFmt := "ot:after_%s"

	// reference: http://gorm.io/docs/write_plugins.html

	// create
	name := create.ToString()
	db.gorm.Callback().Create().Before(fmt.Sprintf(callbackFmt, name)).Register(
		fmt.Sprintf(beforeFmt, name), db.beforeCreate)
	db.gorm.Callback().Create().After(fmt.Sprintf(callbackFmt, name)).Register(
		fmt.Sprintf(afterFmt, name), db.afterCallback)
	// update
	name = update.ToString()
	db.gorm.Callback().Update().Before(fmt.Sprintf(callbackFmt, name)).Register(
		fmt.Sprintf(beforeFmt, name), db.beforeUpdate)
	db.gorm.Callback().Update().After(fmt.Sprintf(callbackFmt, name)).Register(
		fmt.Sprintf(afterFmt, name), db.afterCallback)
	// delete
	name = delete.ToString()
	db.gorm.Callback().Delete().Before(fmt.Sprintf(callbackFmt, name)).Register(
		fmt.Sprintf(beforeFmt, name), db.beforeDelete)
	db.gorm.Callback().Delete().After(fmt.Sprintf(callbackFmt, name)).Register(
		fmt.Sprintf(afterFmt, name), db.afterCallback)
	// query
	name = query.ToString()
	db.gorm.Callback().Query().Before(fmt.Sprintf(callbackFmt, name)).Register(
		fmt.Sprintf(beforeFmt, name), db.beforeQuery)
	db.gorm.Callback().Query().After(fmt.Sprintf(callbackFmt, name)).Register(
		fmt.Sprintf(afterFmt, name), db.afterCallback)
	// rowQuery
	name = rowQuery.ToString()
	db.gorm.Callback().RowQuery().Before(fmt.Sprintf(callbackFmt, name)).Register(
		fmt.Sprintf(beforeFmt, name), db.beforeRowQuery)
	db.gorm.Callback().RowQuery().After(fmt.Sprintf(callbackFmt, name)).Register(
		fmt.Sprintf(afterFmt, name), db.afterCallback)
}

// beforeCallback is before callback function.
func (db *wrapDB) beforeCallback(scope *gorm.Scope, cn callbackName) {
	v, ok := scope.Get(gormCtx)
	if !ok {
		return
	}
	ctx := v.(context.Context)

	driver := db.gorm.Dialect().GetName()
	op := "gorm:" + driver + ":" + strings.ToLower(cn.ToString())

	span, newCtx := opentracing.StartSpanFromContext(ctx, op)
	ext.DBType.Set(span, "sql")
	ext.DBInstance.Set(span, driver)
	ext.SpanKind.Set(span, "client/server")

	scope.Set(gormCtx, newCtx)
}

// afterCallback is before callback function.
func (db *wrapDB) afterCallback(scope *gorm.Scope) {
	v, ok := scope.Get(gormCtx)
	if !ok {
		return
	}
	ctx := v.(context.Context)

	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		method := strings.Split(strings.TrimSpace(scope.SQL), " ")[0]

		// scope.SQL is only exist after query is executed.
		ext.DBStatement.Set(span, strings.ToUpper(scope.SQL))
		span.SetTag("db.table", scope.TableName())
		span.SetTag("db.method", method)

		// if exec is raised an error
		if scope.DB().Error != nil {
			ext.Error.Set(span, true)
			span.LogFields(log.Error(scope.DB().Error))
		}
		span.SetTag("db.rows_affected", scope.DB().RowsAffected)
		span.Finish()
	}
}

// WrapDB returns a wrapping DB which is added callback functions for opentracing.
func WrapDB(db *gorm.DB) (DB, error) {
	if db == nil {
		return nil, fmt.Errorf("[err] WrapDB empty params")
	}
	wdb := &wrapDB{gorm: db}
	// add callback functions to *gorm.DB
	wdb.registerCallbacks()
	return wdb, nil
}
