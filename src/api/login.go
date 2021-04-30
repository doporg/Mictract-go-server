package api

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"math/rand"
	"mictract/config"
	"mictract/dao"
	"mictract/enum"
	"mictract/model"
	"mictract/model/response"
	"net/http"
	"strconv"
	"time"
)

// POST /api/login
func Login(c *gin.Context)  {
	reqInfo := struct {
		UserID		int			`form:"userID" json:"userID" binding:"required"`
		Password	string		`form:"password" json:"password" binding:"required"`
	}{}
	if err := c.ShouldBindJSON(&reqInfo); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	user := &model.CaUser{}
	var err error

	if reqInfo.UserID == config.Super_User_ID {
		if reqInfo.Password == config.Super_User_PW {
			user.ID = reqInfo.UserID
			user.Password = reqInfo.Password
		} else {
			response.Err(http.StatusBadRequest, enum.CodeErrNotFound).
				SetMessage("wrong user name or password").
				Result(c.JSON)
			return
		}
	} else {
		user, err = dao.FindCaUserByID(reqInfo.UserID)
		if err != nil {
			response.Err(http.StatusBadRequest, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
		if user.Password != reqInfo.Password {
			response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
				SetMessage("wrong user name or password").
				Result(c.JSON)
			return
		}
	}

	// session_id = hash(userID, password, randomNumber, expireTime = time.Now + 24 * 60 * 60)
	randomNumber 	:= strconv.Itoa(rand.New(rand.NewSource(time.Now().UnixNano())).Intn(99999999))
	expireTime 		:= time.Now().Unix() + int64(24 * 60 * 60)
	sessionID 		:= compute_session_id(reqInfo.UserID, reqInfo.Password, randomNumber, expireTime)


	http.SetCookie(c.Writer, &http.Cookie{Name: "session_id", Value: sessionID,})
	http.SetCookie(c.Writer, &http.Cookie{Name: "expire_time", Value: strconv.FormatInt(expireTime, 10),})
	http.SetCookie(c.Writer, &http.Cookie{Name: "random_number", Value: randomNumber,})
	http.SetCookie(c.Writer, &http.Cookie{Name: "user_id", Value: strconv.Itoa(reqInfo.UserID),})

	response.Ok().Result(c.JSON)
}

func AuthMiddleWare (c *gin.Context) {
	if url := c.Request.URL.String(); url == "/api/login/" {
		c.Next()
		return
	}

	var userID 			int
	var password		string
	var randomNumber 	string
	var	expireTime 		int64
	var sessionID		string

	if cookie, err := c.Request.Cookie("expire_time"); err == nil {
		expireTime, err = strconv.ParseInt(cookie.Value, 10, 64)
		if err != nil {
			response.Err(http.StatusBadRequest, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
	} else {
		response.Err(http.StatusBadRequest, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		c.Abort()
		return
	}
	if cookie, err := c.Request.Cookie("session_id"); err == nil {
		sessionID = cookie.Value
	} else {
		response.Err(http.StatusBadRequest, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		c.Abort()
		return
	}
	if cookie, err := c.Request.Cookie("user_id"); err == nil {
		userID, err = strconv.Atoi(cookie.Value)
		if err != nil {
			response.Err(http.StatusBadRequest, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			c.Abort()
			return
		}
	} else {
		response.Err(http.StatusBadRequest, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		c.Abort()
		return
	}
	if cookie, err := c.Request.Cookie("random_number"); err == nil {
		randomNumber = cookie.Value
	} else {
		response.Err(http.StatusBadRequest, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		c.Abort()
		return
	}

	if expireTime < time.Now().Unix() {
		response.Err(http.StatusBadRequest, enum.CodeErrNotFound).
			SetMessage("Cookies have expired, please login again").
			Result(c.JSON)
		c.Abort()
		return
	}

	// get password
	if userID == config.Super_User_ID {
		password = config.Super_User_PW
	} else {
		if user, err := dao.FindCaUserByID(userID); err != nil {
			response.Err(http.StatusBadRequest, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			c.Abort()
			return
		} else {
			password = user.Password
		}
	}

	if sessionID != compute_session_id(userID, password, randomNumber, expireTime) {
		fmt.Println(userID)
		fmt.Println(password)
		fmt.Println(randomNumber)
		fmt.Println(expireTime)
		response.Err(http.StatusBadRequest, enum.CodeErrNotFound).
			SetMessage("Authentication_failed").
			Result(c.JSON)
		c.Abort()
		return
	}

	// 根据路由判断一下
	if c.Request.Method == http.MethodPost && c.Request.URL.String() != "/api/chaincode/transaction/" {
		// 必须 super user
		if userID != config.Super_User_ID {
			response.Err(http.StatusUnauthorized, enum.CodeErrNotFound).
				SetMessage("Not enough permissions").
				Result(c.JSON)
			c.Abort()
			return
		}
	}

	c.Set("userID", userID)
	c.Next()
	return
}

func compute_session_id(userID int, password, randomNumber string, expireTime int64) string {
	Md5Inst := md5.New()
	Md5Inst.Write([]byte(strconv.Itoa(userID) + password + randomNumber + strconv.FormatInt(expireTime, 10)))
	return hex.EncodeToString(Md5Inst.Sum([]byte("")))
}


