package init

import (
	"fmt"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"mictract/config"
	"mictract/global"
	"mictract/model"
)

func initDB() {
	mysqlConfig := mysql.Config{
		DSN: fmt.Sprintf("root:%s@tcp(%s:3306)/gorm?charset=utf8&parseTime=True&loc=Local",
			config.DB_PW, config.DB_SERVER_URL),
	}

	var err error
	if global.DB, err =
		gorm.Open(
			mysql.New(mysqlConfig),
			&gorm.Config{
				//Logger: logger.Default.LogMode(logger.Info),
			},
			); err != nil {
		global.Logger.Error("Get database error", zap.Error(err))
	}
}

func closeDB() {
	db, _ := global.DB.DB()
	_ = db.Close()
}

func createTables() {
	err := global.DB.AutoMigrate(
		model.Network{},
		model.Channel{},
		model.Organization{},
		model.CaUser{},
		model.Chaincode{},
	)

	if err != nil {
		global.Logger.Error("create tables failed", zap.Error(err))
	} else {
		global.Logger.Info("tables created")
	}
}
