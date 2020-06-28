# opentracing gorm

[OpenTracing](http://opentracing.io/) instrumentation for [GORM V2](https://github.com/go-gorm/gorm).

## Install

```
go get -u github.com/lhypj/opentracing-gorm
```

## Usage

1. Call `otgorm.AddGormCallbacks(db)` with an instance of your `*gorm.DB`.
2. Clone db `db = otgorm.SetSpanToGorm(ctx, db)` with a span.

Example:

```go
var gDB *gorm.DB

func init() {
    gDB = initDB()
}

func initDB() *gorm.DB {
    instance, err := gorm.Open(mysql.New(mysql.Config{DSN: dsn}), &gorm.Config{})
    if err != nil {
        panic(err)
    }
    // register callbacks must be called for a root instance of your gorm.DB
    otgorm.AddGormCallbacks(db)
    return db
}

func Handler(ctx context.Context) {
    span, ctx := opentracing.StartSpanFromContext(ctx, "handler")
    defer span.Finish()

    // clone db with proper context
    db := otgorm.SetSpanToGorm(ctx, gDB)

    // sql query
    db.First
}
```

Call to the `Handler` function would create sql span with table name, sql method and sql statement as a child of handler span.

## License

[MIT](LICENSE)
