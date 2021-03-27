package model

import (
	"crypto/md5"
	"fmt"
	"io"
	"mictract/global"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type User struct {
	ID 		 	int		`json:"id" gorm:"primarykey"`
	Nickname 	string 	`json:"nickname" binding:"required"`
	OrgID	 	int 	`json:"org_id"`
	NetID		int 	`json:"net_id"`
	UserType 	string 	`json:"usertype"`
	Password	string 	`json:"password"`

	CreatedAt time.Time      `json:"createat"`
	UpdatedAt time.Time      `json:"updateat"`
}

func NewUser(orgID, netID int, nickname, userType, password string) (*User, error) {
	now := time.Now()
	h := md5.New()
	io.WriteString(h, password)
	io.WriteString(h, strconv.Itoa(now.Nanosecond()))
	u := &User{
		OrgID: orgID,
		NetID: netID,
		Nickname: nickname,
		UserType: userType,
		Password: fmt.Sprintf("%x", h.Sum(nil)),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := global.DB.Create(u).Error; err != nil {
		return nil, errors.WithMessage(err, "Unable to insert user")
	}
	return u, nil
}

func DelUser(userID int) error {
	if err := global.DB.Where("id = ?", userID).Delete(&User{}).Error; err != nil {
		return errors.WithMessage(err, "Unable to delete user")
	}
	return nil
}


func QueryAllUser() ([]User, error) {
	users := []User{}
	if err := global.DB.Find(&users).Error; err != nil {
		return nil, errors.WithMessage(err, "Fail to query all users")
	}
	return users, nil
}