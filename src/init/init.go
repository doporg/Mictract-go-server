package init

import (
	"mictract/global"
	"mictract/model"
)

func init() {
	// initialization code goes here.
	createTables()
	initNetsAndSDKs()
}

func Close() {
	model.UpsertAllNets()
	//for _, net := range global.Nets {
	//	n := net.(model.Network)
	//	if err := n.Insert(); err != nil {
	//		global.Logger.Error("fail to insert net ", zap.Error(err))
	//	}
	//}
	global.Close()
}