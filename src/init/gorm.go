package init

// If you want to initialize gorm, uncomment the code below.

import (
	"mictract/global"
	"mictract/model"

	"go.uber.org/zap"
)

func createTables() {
	err := global.DB.AutoMigrate(
		model.User{},
		model.Network{},
	)



	if err != nil {
		global.Logger.Error("create tables failed", zap.Error(err))
	} else {
		global.Logger.Info("tables created")
	}
}

func initNetsAndSDKs() {
	nets, err := model.QueryAllNetwork()
	if err != nil {
		global.Logger.Error("fail to get all network from db", zap.Error(err))
	}
	for _, net := range nets {
		model.UpdateNets(net)
		// model.UpdateSDK(net.ID)
	}
}

// func createUsers() {
// 	var users = []model.User{
// 		{
// 			Model: gorm.Model{
// 				ID:        1,
// 				CreatedAt: time.Now(),
// 				UpdatedAt: time.Now(),
// 				DeletedAt: gorm.DeletedAt{},
// 			}, Name: "zhangsan", Age: 12,
// 		},
// 		{
// 			Model: gorm.Model{
// 				ID:        2,
// 				CreatedAt: time.Now(),
// 				UpdatedAt: time.Now(),
// 				DeletedAt: gorm.DeletedAt{},
// 			}, Name: "lisi", Age: 20,
// 		},
// 	}

// if global.DB.Where("id in ?", []int{1, 2}).Find(&[]model.User{}).RowsAffected == 2 {
// 	global.Logger.Info("user data has been initialized")
// } else if err := global.DB.Create(&users).Error; err != nil {
// 	global.Logger.Error("user data initialize error", zap.Error(err))
// } else {
// 	global.Logger.Info("user data initialized")
// }
// }
