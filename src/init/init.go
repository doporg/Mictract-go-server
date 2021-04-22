package init

import (
	"mictract/global"
	"mictract/model/kubernetes"
	"mictract/service"
)

func init() {
	_ = (&kubernetes.Tools{}).AwaitableCreate()
	initDB()
	createTables()

	service.StartMyAlert()
}

func Close() {
	closeDB()
	(&kubernetes.Tools{}).Delete()
	global.Close()
}