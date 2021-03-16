package init

import (
	"mictract/global"
	"mictract/model"
	"mictract/model/kubernetes"
	"time"
)

func init() {
	// initialization code goes here.
	(&kubernetes.Tools{}).Create()
	(&kubernetes.Mysql{}).Create()
	// TODO: sync
	time.Sleep(5 * time.Second)

	createTables()
	initNetsAndSDKs()
}

func Close() {
	model.UpsertAllNets()

	(&kubernetes.Tools{}).Delete()
	(&kubernetes.Mysql{}).Delete()
	//for _, net := range global.Nets {
	//	n := net.(model.Network)
	//	if err := n.Insert(); err != nil {
	//		global.Logger.Error("fail to insert net ", zap.Error(err))
	//	}
	//}
	global.Close()
}