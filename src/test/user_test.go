package test

import (
	"fmt"
	"mictract/global"
	"mictract/model"
	"testing"

	"go.uber.org/zap"
)

func TestCURD(t *testing.T) {
	createTables()

	for _, pair := range [][]string{
		[]string{"zhangsan", "Admin@org2.net1.com"},
		[]string{"zhangsan", "User1@net1.com"},
		[]string{"zhangsan", "User100@org2.net1.com"},
		[]string{"lisi", "Admin@org1.net2.com"},
		[]string{"wangwu", "User@org1000.net1422.com"},
	} {
		if err := model.AddUser(pair[0], pair[1]); err != nil {
			global.Logger.Fatal("fail to add", zap.Error(err))
		}
	}

	show()

	if err := model.DelUser("User@org1000.net1422.com"); err != nil {
		global.Logger.Fatal("fail to del", zap.Error(err))
	}

	show()

	if err := model.UpdateNickName("User1@net1.com", "zhangsan", "zhaoliu"); err != nil {
		global.Logger.Fatal("fail to update", zap.Error(err))
	}

	show()

	users, err := model.QueryUser("zhangsan")
	if err != nil {
		global.Logger.Fatal("fail to query", zap.Error(err))
	}
	fmt.Println(users)

}

func show() {
	users, err := model.QueryAllUser()
	if err != nil {
		global.Logger.Fatal("fail to query all", zap.Error(err))
	}
	fmt.Println(users)
}

func createTables() {
	err := global.DB.AutoMigrate(
		model.User{},
	)

	if err != nil {
		global.Logger.Error("create tables failed", zap.Error(err))
	} else {
		global.Logger.Info("tables created")
	}
}
