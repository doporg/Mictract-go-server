package model

import (
	"fmt"
	"go.uber.org/zap"
	mConfig "mictract/config"
	"mictract/global"
	"os"
	"path"
	"path/filepath"
	"text/template"
	"time"
)

type Network struct {
	ID        	int 		`json:"id" gorm:"primarykey"`
	Nickname	string 		`json:"nickname"`
	CreatedAt 	time.Time
	Status 		string 		`json:"status"`

	Consensus  	string 		`json:"consensus" binding:"required"`
	TlsEnabled 	bool   		`json:"tlsEnabled"`
}

func GetNetworkNameByID(netID int) string {
	return fmt.Sprintf("net%d", netID)
}
func (n *Network) GetName() string {
	return GetNetworkNameByID(n.ID)
}

func (n *Network) RemoveAllFile() {
	if err := os.RemoveAll(filepath.Join(mConfig.LOCAL_BASE_PATH, GetNetworkNameByID(n.ID))); err != nil {
		global.Logger.Error("fail to remove all file", zap.Error(err))
	}
}

// only for OrdererOrg
func (orderer *CaUser) RenderConfigtx() error {
	templ := template.Must(template.ParseFiles(path.Join(mConfig.LOCAL_MOUNT_PATH, "configtx.yaml.tpl")))

	filename := fmt.Sprintf("/mictract/networks/net%d/configtx.yaml", orderer.NetworkID)
	writer, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer writer.Close()

	if err := templ.Execute(writer, orderer); err != nil {
		return err
	}

	return nil
}
