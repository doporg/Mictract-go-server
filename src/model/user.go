package model

import (
	"mictract/global"
	"strings"
	"time"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type User struct {
	Nickname string `json:"nickname" binding:"required"`
	Username string `json:"username" gorm:"primarykey" binding:"required"`
	// Admin@net1.com Admin@org1.net2.com User5@org6.net3.com

	CreatedAt time.Time      `json:"createat"`
	UpdatedAt time.Time      `json:"updateat"`
	DeletedAt gorm.DeletedAt `json:"deletedat" gorm:"index"`

	UserType string `json:"usertype"`
	OrgName  string `json:"orgname"`
	NetName  string `json:"netname"`
}

func AddUser(Nickname, Username string) error {
	user := &User{
		Nickname:  Nickname,
		Username:  Username,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		DeletedAt: gorm.DeletedAt{},
	}
	user.parse()

	if err := global.DB.Create(&user).Error; err != nil {
		return errors.WithMessage(err, "Unable to insert user")
	}
	return nil
}

func DelUser(username string) error {
	if err := global.DB.Where("username = ?", username).Delete(&User{}).Error; err != nil {
		return errors.WithMessage(err, "Unable to delete user")
	}
	return nil
}

func QueryUserByNickName(nickname string) ([]User, error) {
	users := []User{}
	if err := global.DB.Where("nickname = ?", nickname).Find(&users).Error; err != nil {
		return nil, errors.WithMessage(err, "Fail to query")
	}
	return users, nil
}

func QueryUserByUserName(username string) ([]User, error) {
	users := []User{}
	if err := global.DB.Where("username = ?", username).Find(&users).Error; err != nil {
		return nil, errors.WithMessage(err, "Fail to query")
	}
	return users, nil
}

func QueryAllUser() ([]User, error) {
	users := []User{}
	if err := global.DB.Find(&users).Error; err != nil {
		return nil, errors.WithMessage(err, "Fail to query all users")
	}
	return users, nil
}

func UpdateNickName(username, oldNickname, newNickname string) error {
	users, err := QueryUserByNickName(oldNickname)
	if err != nil {
		return errors.WithMessage(err, "No such user found")
	}

	isExist := false
	for _, user := range users {
		if user.Username == username {
			isExist = true
			break
		}
	}
	if !isExist {
		return errors.New("No such user found")
	}

	if err := global.DB.Model(&User{}).Where("username = ?", username).Updates(User{Nickname: newNickname, UpdatedAt: time.Now()}).Error; err != nil {
		return errors.WithMessage(err, "Fail to update")
	}

	return nil
}

func (u *User) parse() {
	if strings.HasPrefix(u.Username, "Admin") {
		u.UserType = "admin"
	} else {
		u.UserType = "user"
	}
	tmps := strings.Split(strings.Split(u.Username, "@")[1], ".")
	if strings.Contains(u.Username, "org") {
		u.OrgName = tmps[0]
		u.NetName = tmps[1]
	} else {
		u.OrgName = "ordererorg"
		u.NetName = tmps[0]
	}
}
