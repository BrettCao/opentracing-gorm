package otgorm

import (
	"context"
	"fmt"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jinzhu/gorm"
	"github.com/opentracing/opentracing-go/ext"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

func TestWrapDB(t *testing.T) {
	assert := assert.New(t)

	db, _, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	gdb, err := gorm.Open("allan-test", db)
	if err != nil {
		panic(err)
	}

	tests := map[string]struct {
		db    *gorm.DB
		isErr bool
	}{
		"fail":    {db: nil, isErr: true},
		"success": {db: gdb},
	}

	for k, tc := range tests {
		t.Run(k, func(t *testing.T) {
			_, err := WrapDB(tc.db)
			assert.Equal(tc.isErr, err != nil)
		})
	}
}

func TestWrapDB_WithContext(t *testing.T) {
	assert := assert.New(t)

	db, _, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	gdb, err := gorm.Open("allan-test", db)
	if err != nil {
		panic(err)
	}
	wdb, err := WrapDB(gdb)
	if err != nil {
		panic(err)
	}

	tests := map[string]struct {
		db  DB
		ctx context.Context
	}{
		"check": {db: wdb, ctx: context.TODO()},
	}

	for k, tc := range tests {
		t.Run(k, func(t *testing.T) {
			newdb := tc.db.WithContext(tc.ctx)
			v, ok := newdb.Get(gormCtx)
			assert.True(ok)
			assert.Equal(tc.ctx, v)
		})
	}
}

func TestWrapDB_Callback(t *testing.T) {
	assert := assert.New(t)

	opentracing.SetGlobalTracer(mocktracer.New())

	db, _, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	gdb, err := gorm.Open("common", db)
	if err != nil {
		panic(err)
	}
	wdb, err := WrapDB(gdb)
	if err != nil {
		panic(err)
	}

	tests := map[string]struct {
		db *wrapDB
	}{
		"success": {db: wdb.(*wrapDB)},
	}

	for k, tc := range tests {
		t.Run(k, func(t *testing.T) {
			switch k {
			case "success":
				err := fmt.Errorf("error test")
				db := tc.db.WithContext(context.TODO())
				scope := db.Table("users").Select("id").NewScope(db.Value)
				scope.DB().Error = err

				tc.db.beforeQuery(scope)
				tc.db.afterCallback(scope)

				v, _ := scope.Get(gormCtx)
				ctx := v.(context.Context)
				span := opentracing.SpanFromContext(ctx).(*mocktracer.MockSpan)

				assert.Equal("gorm:common:query", span.OperationName)
				for k, v := range span.Tags() {
					switch k {
					case string(ext.DBType):
						assert.Equal("sql", v.(string))
					case string(ext.DBInstance):
						assert.Equal("common", v.(string))
					case string(ext.SpanKind):
						assert.Equal("client/server", string(v.(ext.SpanKindEnum)))
					case string(ext.DBStatement), "db.method":
						// don't check
					case "db.table":
						assert.Equal("users", v)
					case "db.rows_affected":
						assert.Equal(int64(0), v.(int64))
					case "error":
						assert.Equal(true, v.(bool))
					default:
						panic("unknown tag")
					}
				}
				for _, v := range span.Logs() {
					assert.Equal(err.Error(), v.Fields[0].ValueString)
				}
			}
		})
	}
}
