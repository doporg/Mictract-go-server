package global

// If you would like to use sql db, uncomment the code below.

import (
	"fmt"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"mictract/config"
)

func initDB() {
	mysqlConfig := mysql.Config{
		// DSN: "root:123456@tcp(127.0.0.1:3306)/gorm?charset=utf8&parseTime=True&loc=Local",
		DSN: fmt.Sprintf("root:%s@tcp(mysql:3306)/gorm?charset=utf8&parseTime=True&loc=Local", config.MYSQL_PW),
	}

	var err error
	if DB, err = gorm.Open(mysql.New(mysqlConfig), &gorm.Config{}); err != nil {
		Logger.Error("Get database error", zap.Error(err))
	}
}

func closeDB() {
	db, _ := DB.DB()
	_ = db.Close()
}
