package global

// If you would like to use sql db, uncomment the code below.

// import (
// 	"go.uber.org/zap"
// 	"gorm.io/driver/mysql"
// 	"gorm.io/gorm"
// )
//
// func initDB() {
// 	mysqlConfig := mysql.Config{
// 		DSN: "root:1234@tcp(127.0.0.1:3306)/gorm?charset=utf8&parseTime=True&loc=Local",
// 	}
//
// 	var err error
// 	if DB, err = gorm.Open(mysql.New(mysqlConfig), &gorm.Config{}); err != nil {
// 		Logger.Error("Get database error", zap.Error(err))
// 	}
// }
//
// func closeDB() {
// 	db, _ := DB.DB()
// 	_ = db.Close()
// }
//
//