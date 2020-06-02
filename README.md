# opentracing-gorm
<p align="left">
<a href="https://hits.seeyoufarm.com"/><img src="https://hits.seeyoufarm.com/api/count/incr/badge.svg?url=https%3A%2F%2Fgithub.com%2FBrettCao%2Fopentracing-gorm"/></a>
<a href="https://goreportcard.com/report/github.com/BrettCao/opentracing-gorm"><img src="https://goreportcard.com/badge/github.com/BrettCao/opentracing-gorm" alt="Go Report Card" /></a> 
<a href="/LICENSE"><img src="https://img.shields.io/badge/license-MIT-GREEN.svg" alt="license" /></a>
</p>

This project is a [opentracing](http://opentracing.io/) wrapper for [gorm](https://github.com/jinzhu/gorm).

## Getting Started
```go
// pseudo code
package main
import (
	"net/http"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	otgorm "github.com/BrettCao/opentracing-gorm"
)

func main() {
	// TODO: Set global opentracing tracer
	db, _ := gorm.Open("mysql", "user:password@/dbname")
	wdb, _ := otgorm.WrapDB(db)
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		gormDB := wdb.WithContext(r.Context())
		_ = gormDB.Table("users").Select("id").Row()
	})
	http.ListenAndServe(":8080", nil)
}
```
 
## License
This project is licensed under the MIT License
