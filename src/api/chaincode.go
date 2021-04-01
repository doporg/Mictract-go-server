package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	channelclient "github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"go.uber.org/zap"
	"mictract/enum"
	"mictract/global"
	"mictract/model"
	"mictract/model/request"
	"mictract/model/response"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
)

// POST /api/chaincode

func CreateChaincode(c *gin.Context)  {
	// 如果这里数组越界，应该时网络创建时的问题
	// 1. upload
	// 2. unpack
	// 3. build
	// 4. install (all peers)
	// 5. approve (channel's org)
	// 6. commmit
	// 7. start cc container
	var (
		// ccType 			= c.PostForm("ccType")
		nickname 		= c.PostForm("nickname")

		label 			= c.PostForm("label")
		policyStr 		= c.PostForm("policy")
		version			= c.PostForm("version")
		sequence		= c.PostForm("sequence")
		initRequired	= c.PostForm("initRequired")

		channelName		= c.PostForm("channelName")
		NetworkURL		= c.PostForm("network")
	)
	ccType := "go"

	srcTarGz, err := c.FormFile("file")
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	var cc *model.Chaincode
	var cci *model.ChaincodeInstance
	if cc, err = model.NewChaincode(nickname, ccType); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	var channelID int
	IdExp := regexp.MustCompile("^(channel)([0-9]+)$")
	if matches := IdExp.FindStringSubmatch(channelName); len(matches) < 2 {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage("Error occurred in matching channelID").
			Result(c.JSON)
		return
	} else {
		channelID, _ = strconv.Atoi(matches[2])
	}
	netID := model.NewCaUserFromDomainName(NetworkURL).NetworkID
	_sequence, _ := strconv.Atoi(sequence)
	_initReq, _ := strconv.ParseBool(initRequired)
	if cci, err = cc.NewChaincodeInstance(
		netID,
		channelID,
		label,
		"",
		policyStr,
		version,
		int64(_sequence),
		true,
		_initReq,
		); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	err = c.SaveUploadedFile(srcTarGz, filepath.Join(cc.GetCCPath(), "src.tar.gz"))
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	go func() {
		// 2. unpack
		if err := cc.Unpack(); err != nil {
			global.Logger.Error("fail to unpack cc", zap.Error(err))
			cc.Status = "error"
			model.UpdateChaincode(cc)
			return
		}

		// 3. build
		cc.Status = "building"
		model.UpdateChaincode(cc)
		if err := cci.Build(); err != nil {
			global.Logger.Error("fail to build cc ", zap.Error(err))
			cc.Status = "error"
			model.UpdateChaincode(cc)
			return
		}

		// 4. install (all peers)
		global.Logger.Info("Obtaining sdk...")
		model.UpdateSDK(netID)
		sdk, err := model.GetSDKByNetWorkID(netID)
		if err != nil {
			global.Logger.Error("fail to get sdk", zap.Error(err))
			cc.Status = "error"
			model.UpdateChaincode(cc)
			return
		}

		cc.Status = "installing"
		model.UpdateChaincode(cc)
		ch, err := model.GetChannelFromNets(channelID, netID)
		if err != nil {
			global.Logger.Error("fail to get ch", zap.Error(err))
			cc.Status = "error"
			model.UpdateChaincode(cc)
			return
		}
		global.Logger.Info(fmt.Sprintf("%v", ch))
		for _, org := range ch.Organizations {
			global.Logger.Info("Obtaining rc...")
			rc, err := resmgmt.New(
				sdk.Context(
					fabsdk.WithUser(fmt.Sprintf("Admin1@org%d.net%d.com", org.ID, org.NetworkID)),
					fabsdk.WithOrg(fmt.Sprintf("org%d", org.ID))))
			if err != nil {
				global.Logger.Error("fail to get rc", zap.Error(err))
				cc.Status = "error"
				model.UpdateChaincode(cc)
				return
			}

			if err := cci.InstallCC(rc); err != nil {
				global.Logger.Error(fmt.Sprintf("fail to install cc to org%d", org.ID), zap.Error(err))
				cc.Status = "error"
				model.UpdateChaincode(cc)
				return
			}
		}

		// 5. approve (channel's org)
		for _, org := range ch.Organizations {
			global.Logger.Info("Obtaining rc...")
			rc, err := resmgmt.New(
				sdk.Context(
					fabsdk.WithUser(fmt.Sprintf("Admin1@org%d.net%d.com", org.ID, org.NetworkID)),
					fabsdk.WithOrg(fmt.Sprintf("org%d", org.ID))))
			if err != nil {
				global.Logger.Error("fail to get rc", zap.Error(err))
				cc.Status = "error"
				model.UpdateChaincode(cc)
				return
			}

			if err := cci.ApproveCC(rc, fmt.Sprintf("orderer1.net%d.com", org.NetworkID), org.Peers[0].Name); err != nil {
				global.Logger.Error("fail to get approve cc", zap.Error(err))
				cc.Status = "error"
				model.UpdateChaincode(cc)
				return
			}

			cci.CheckCCCommitReadiness(rc)
		}


		// 6. commmit
		net, err := model.GetNetworkfromNets(netID)
		if err != nil {
			global.Logger.Error("fail to get net", zap.Error(err))
			cc.Status = "error"
			model.UpdateChaincode(cc)
			return
		}
		peerURLs := []string{}
		for _, org := range net.Organizations {
			if org.ID == -1 {
				continue
			}
			for _, peer := range org.Peers {
				peerURLs = append(peerURLs, peer.Name)
			}
		}

		global.Logger.Info("Obtaining rc...")
		org := ch.Organizations[0]
		rc, err := resmgmt.New(
			sdk.Context(
				fabsdk.WithUser(fmt.Sprintf("Admin1@org%d.net%d.com", org.ID, org.NetworkID)),
				fabsdk.WithOrg(fmt.Sprintf("org%d", org.ID))))
		if err != nil {
			global.Logger.Error("fail to get rc", zap.Error(err))
			cc.Status = "error"
			model.UpdateChaincode(cc)
			return
		}

		if err := cci.CommitCC(rc, fmt.Sprintf("orderer1.net%d.com", org.NetworkID), peerURLs...); err != nil {
			global.Logger.Error("fail to commit cc", zap.Error(err))
			cc.Status = "error"
			model.UpdateChaincode(cc)
			return
		}

		// 7. start cc container
		if err := cci.CreateEntity(); err != nil {
			global.Logger.Error("fail to create cc container", zap.Error(err))
			cc.Status = "error"
			model.UpdateChaincode(cc)
			return
		}

		cc.Status = "running"
		model.UpdateChaincode(cc)

	}()

	response.Ok().
		Result(c.JSON)
}

// POST /api/chaincode/:id
func InvokeChaincode(c *gin.Context)  {
	ccID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	var info request.InvokeCCReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	nets, err := model.QueryAllNetwork()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	var cci model.ChaincodeInstance
	var channel model.Channel
	flag := false
	for _, net := range nets {
		for _, ch := range net.Channels {
			for _, cc := range ch.Chaincodes {
				if cc.CCID == ccID {
					flag = true
					cci  = cc
					channel = ch
				}
			}
		}
	}
	if !flag {
		response.Err(http.StatusBadRequest, enum.CodeErrNotFound).
			SetMessage("Chaincode not found").
			Result(c.JSON)
		return
	}

	global.Logger.Info("Obtaining sdk...")
	if err := model.UpdateSDK(cci.NetworkID); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	sdk, err := model.GetSDKByNetWorkID(cci.NetworkID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	global.Logger.Info("Obtaining channel client...")
	ccp := sdk.ChannelContext(
		fmt.Sprintf("channel%d", cci.ChannelID),
		fabsdk.WithUser(fmt.Sprintf("Admin1@org%d.net%d.com", channel.Organizations[0].ID, channel.NetworkID)),
		fabsdk.WithOrg(fmt.Sprintf("org%d", channel.Organizations[0].ID)))
	chClient, err := channelclient.New(ccp)
	if err != nil {
		global.Logger.Error("fail to get channel client", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	var ret channelclient.Response
	switch info.InvokeType {
	case "init":
		ret, err = cci.InitCC(chClient, info.Args, info.PeerURLs...)
	case "query":
		ret, err = cci.QueryCC(chClient, info.Args, info.PeerURLs...)
	case "execute":
		ret, err = cci.ExecuteCC(chClient, info.Args, info.PeerURLs...)
	default:
		response.Err(http.StatusBadRequest, enum.CodeErrBadArgument).
			SetMessage("invokeType only supports init, execute, query").
			Result(c.JSON)
		return
	}
	if err != nil {
		global.Logger.Error(err.Error())
		response.Err(http.StatusInternalServerError, enum.CodeErrBlockchainNetworkError).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	response.Ok().
		SetPayload(ret).
		Result(c.JSON)
}

// DELETE /api/chaincode
func DeleteChaincode(c *gin.Context)  {
	var info request.DelCCReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := model.DeleteChaincodeByID(info.CCID); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		Result(c.JSON)
}

// PATCH /api/chaincode
func UpdateChaincodeNickName(c *gin.Context) {
	var info request.UpdateCCNickNameReq
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := model.UpdateChaincodeNickname(info.CCID, info.NewNickname); err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().Result(c.JSON)
}

// GET /api/chaincode
// ChaincodeInstance
func ListChaincodes(c *gin.Context) {
	//ccs, err := model.ListAllChaincodes()
	//if err != nil {
	//	response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
	//		SetMessage(err.Error()).
	//		Result(c.JSON)
	//	return
	//}
	ccis := []model.ChaincodeInstance{}
	nets, err := model.QueryAllNetwork()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	for _, net := range nets {
		for _, ch := range net.Channels {
			ccis = append(ccis, ch.Chaincodes...)
		}
	}

	response.Ok().
		SetPayload(response.NewChaincodes(ccis)).
		Result(c.JSON)
}