package otgorm_test

import (
	"context"
	"log"
	"testing"

	otgorm "github.com/lhypj/opentracing-gorm"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var tracer *mocktracer.MockTracer

func GetInstance() *gorm.DB {
	var db *gorm.DB
	dsn := "root:zxcvbnm123@tcp(localhost:3306)/testdb?parseTime=True&loc=Asia%2FShanghai"
	db, err := gorm.Open(mysql.New(mysql.Config{DSN: dsn}), &gorm.Config{})
	if err != nil {
		log.Fatalf("open db: %s", err)
	}
	db.AutoMigrate(&Product{})
	db.Create(&Product{Code: "L1212"})
	otgorm.AddGormCallbacks(db)
	return db
}

type Product struct {
	gorm.Model
	Code string
}

func Handler(ctx context.Context) {
	db := GetInstance()
	tracer = mocktracer.New()
	opentracing.SetGlobalTracer(tracer)


	span, ctx := opentracing.StartSpanFromContext(ctx, "handler")
	defer span.Finish()

	db = otgorm.SetSpanToGorm(ctx, db)

	var product Product
	db.WithContext(ctx).Where("id = 1").First(&product)
}

func TestPool(t *testing.T) {
	Handler(context.Background())
	spans := tracer.FinishedSpans()
	if len(spans) != 2 {
		t.Fatalf("should be 2 finished spans but there are %d: %v", len(spans), spans)
	}

	sqlSpan := spans[0]
	if sqlSpan.OperationName != "sql" {
		t.Errorf("first span operation should be sql but it's '%s'", sqlSpan.OperationName)
	}

	expectedTags := map[string]interface{}{
		"error":        false,
		"db.table":     "products",
		"db.method":    "SELECT",
		"db.type":      "sql",
		"db.statement": "SELECT * FROM `products` WHERE id = 1 AND `products`.`deleted_at` IS NULL ORDER BY `products`.`id` LIMIT 1",
		"db.err":       false,
		"db.count":     int64(1),
	}

	sqlTags := sqlSpan.Tags()
	if len(sqlTags) != len(expectedTags) {
		t.Errorf("sql span should have %d tags but it has %d", len(expectedTags), len(sqlTags))
	}

	for name, expected := range expectedTags {
		value, ok := sqlTags[name]
		if !ok {
			t.Errorf("sql span doesn't have tag '%s'", name)
			continue
		}
		if value != expected {
			t.Errorf("sql span tag '%s' should have value '%s' but it has \n'%s'", name, expected, value)
		}
	}

	if spans[1].OperationName != "handler" {
		t.Errorf("second span operation should be handler but it's '%s'", spans[1].OperationName)
	}
}
