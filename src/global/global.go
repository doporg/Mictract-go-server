package global

import (
	"go.uber.org/zap"
)

var (
	// global variables go here.
	// DB 		*gorm.DB
	Logger	*zap.Logger
)

func init() {
	// initDB()
	initLogger()
}

func Close() {
	// closeDB()
}