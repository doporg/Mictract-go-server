package init

import (
	"mictract/global"
	"mictract/model/kubernetes"
)

func init() {
	_ = (&kubernetes.Tools{}).AwaitableCreate()
	initDB()
	createTables()
}

func Close() {
	closeDB()
	(&kubernetes.Tools{}).Delete()
	global.Close()
}