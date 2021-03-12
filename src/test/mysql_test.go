package test

import (
	"mictract/global"
	ii "mictract/init"
	"mictract/model"
	"testing"
)

func TestNetworkCRUD(t *testing.T) {
	global.Logger.Info("start test ....")
	for i := 0; i < 10; i++ {
		model.UpdateNets(*model.GetBasicNetwork())
	}
	model.DeleteNetworkByID(1)
	model.UpdateNets(*model.GetBasicNetwork())
	global.Logger.Info("db will close...")
	ii.Close()
}