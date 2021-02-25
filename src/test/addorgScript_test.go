package test

import (
	"fmt"
	"mictract/config"
	"mictract/global"
	"os/exec"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestABC(t *testing.T) {
	global.Logger.Error("fuck you")
	fmt.Println("abc")
}
func TestAddOrg(t *testing.T) {
	channelID := "mychannel"
	mspID := "Org3MSP"

	cmd := exec.Command(filepath.Join(config.LOCAL_BASE_PATH, "scripts/addorg/addorg.sh"), "addOrg", channelID, mspID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		global.Logger.Error("fail to exec cmd", zap.Error(err))
	}
	fmt.Println(string(output))
}

func TestAddOrderers(t *testing.T) {
	channelID := "byfn-sys-channel"
	mspID := "Org3MSP"

	cmd := exec.Command(filepath.Join(config.LOCAL_BASE_PATH, "scripts/addorg/addorg.sh"), "addOrderers", channelID, mspID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		global.Logger.Error("fail to exec cmd", zap.Error(err))
	}
	fmt.Println(string(output))
}

func TestUpdateAnchors(t *testing.T) {
	channelID := "mychannel"

	cmd := exec.Command(filepath.Join(config.LOCAL_BASE_PATH, "scripts/addorg/addorg.sh"), "addOrderers", channelID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		global.Logger.Error("fail to exec cmd", zap.Error(err))
	}
	fmt.Println(string(output))
}
