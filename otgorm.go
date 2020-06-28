package otgorm

import (
	"context"
	"fmt"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"gorm.io/gorm"
	"strings"
)

const (
	parentSpanGormKey = "opentracingParentSpan"
	spanGormKey       = "opentracingSpan"
)

// SetSpanToGorm sets span to gorm settings, returns cloned DB
func SetSpanToGorm(ctx context.Context, db *gorm.DB) *gorm.DB {
	if ctx == nil {
		return db
	}
	parentSpan := opentracing.SpanFromContext(ctx)
	if parentSpan == nil {
		return db
	}
	return db.Set(parentSpanGormKey, parentSpan)
}

// AddGormCallbacks adds callbacks for tracing, you should call SetSpanToGorm to make them work
func AddGormCallbacks(db *gorm.DB) {
	callbacks := newCallbacks()
	registerCallbacks(db, "create", callbacks)
	registerCallbacks(db, "query", callbacks)
	registerCallbacks(db, "update", callbacks)
	registerCallbacks(db, "delete", callbacks)
	registerCallbacks(db, "row_query", callbacks)
}

type callbacks struct{}

func newCallbacks() *callbacks {
	return &callbacks{}
}

func (c *callbacks) beforeCreate(db *gorm.DB)   { c.before(db) }
func (c *callbacks) afterCreate(db *gorm.DB)    { c.after(db, "INSERT") }
func (c *callbacks) beforeQuery(db *gorm.DB)    { c.before(db) }
func (c *callbacks) afterQuery(db *gorm.DB)     { c.after(db, "SELECT") }
func (c *callbacks) beforeUpdate(db *gorm.DB)   { c.before(db) }
func (c *callbacks) afterUpdate(db *gorm.DB)    { c.after(db, "UPDATE") }
func (c *callbacks) beforeDelete(db *gorm.DB)   { c.before(db) }
func (c *callbacks) afterDelete(db *gorm.DB)    { c.after(db, "DELETE") }
func (c *callbacks) beforeRowQuery(db *gorm.DB) { c.before(db) }
func (c *callbacks) afterRowQuery(db *gorm.DB)  { c.after(db, "") }

func (c *callbacks) before(db *gorm.DB) {
	val, ok := db.Get(parentSpanGormKey)
	if !ok {
		return
	}
	parentSpan := val.(opentracing.Span)
	tr := parentSpan.Tracer()
	sp := tr.StartSpan("sql", opentracing.ChildOf(parentSpan.Context()))
	ext.DBType.Set(sp, "sql")
	db.Set(spanGormKey, sp)
}

func (c *callbacks) after(db *gorm.DB, operation string) {
	val, ok := db.Get(spanGormKey)
	if !ok {
		return
	}
	sql := db.Statement.SQL.String()
	sp := val.(opentracing.Span)
	if operation == "" {
		operation = strings.ToUpper(strings.Split(sql, " ")[0])
	}
	ext.Error.Set(sp, db.Error != nil)
	ext.DBStatement.Set(sp, sql)
	sp.SetTag("db.table", db.Statement.Table)
	sp.SetTag("db.method", operation)
	sp.SetTag("db.err", db.Error != nil)
	sp.SetTag("db.count", db.RowsAffected)
	sp.Finish()
}

func registerCallbacks(db *gorm.DB, name string, c *callbacks) {
	beforeName := fmt.Sprintf("tracing:%v_before", name)
	afterName := fmt.Sprintf("tracing:%v_after", name)
	gormCallbackName := fmt.Sprintf("gorm:%v", name)
	// gorm does some magic, if you pass CallbackProcessor here - nothing works
	switch name {
	case "create":
		db.Callback().Create().Before(gormCallbackName).Register(beforeName, c.beforeCreate)
		db.Callback().Create().After(gormCallbackName).Register(afterName, c.afterCreate)
	case "query":
		db.Callback().Query().Before(gormCallbackName).Register(beforeName, c.beforeQuery)
		db.Callback().Query().After(gormCallbackName).Register(afterName, c.afterQuery)
	case "update":
		db.Callback().Update().Before(gormCallbackName).Register(beforeName, c.beforeUpdate)
		db.Callback().Update().After(gormCallbackName).Register(afterName, c.afterUpdate)
	case "delete":
		db.Callback().Delete().Before(gormCallbackName).Register(beforeName, c.beforeDelete)
		db.Callback().Delete().After(gormCallbackName).Register(afterName, c.afterDelete)
	case "row_query":
		db.Callback().Row().Before(gormCallbackName).Register(beforeName, c.beforeRowQuery)
		db.Callback().Row().Before(gormCallbackName).Register(afterName, c.afterRowQuery)
	}
}
