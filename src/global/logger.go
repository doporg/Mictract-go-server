package global

import (
	"fmt"
	"go.uber.org/zap"
)

func initLogger() {
	var err error
	if Logger, err = zap.NewDevelopment(); err != nil {
		fmt.Printf("Get logger error: %v", err.Error())
	}
}