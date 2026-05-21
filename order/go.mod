module github.com/shijl0925/gin-ninja/order

go 1.26

require (
	github.com/shijl0925/go-toolkits v0.2.3
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	golang.org/x/text v0.21.0 // indirect
)

replace github.com/shijl0925/gin-ninja => ..

replace github.com/shijl0925/gin-ninja/admin => ../admin

replace github.com/shijl0925/gin-ninja/bootstrap => ../bootstrap

replace github.com/shijl0925/gin-ninja/cache/redis => ../cache/redis

replace github.com/shijl0925/gin-ninja/examples => ../examples

replace github.com/shijl0925/gin-ninja/filter => ../filter

replace github.com/shijl0925/gin-ninja/middleware => ../middleware

replace github.com/shijl0925/gin-ninja/orm => ../orm

replace github.com/shijl0925/gin-ninja/pkg/logger => ../pkg/logger

replace github.com/shijl0925/gin-ninja/settings => ../settings
