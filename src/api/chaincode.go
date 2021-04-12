package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	channelclient "github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"go.uber.org/zap"
	"mictract/dao"
	"mictract/enum"
	"mictract/global"
	"mictract/model/request"
	"mictract/model/response"
	"mictract/service"
	"mictract/service/factory"
	"mictract/service/factory/sdk"
	"net/http"
	"path/filepath"
	"strconv"
)

// POST /api/chaincode

func CreateChaincode(c *gin.Context)  {
	// 如果这里数组越界，应该时网络创建时的问题
	// 1. upload
	// 2. unpack
	// 3. build
	// 4. install (one peer)
	// 5. approve (channel's org)
	// 6. commmit
	// 7. start cc container
	var (
		nickname 		= c.PostForm("nickname")

		label 			= c.PostForm("label")
		policyStr 		= c.PostForm("policy")
		version			= c.PostForm("version")
		sequence		= c.PostForm("sequence")
		initRequired	= c.PostForm("initRequired")

		channelID		= c.PostForm("channelID")
	)

	srcTarGz, err := c.FormFile("file")
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	// 2. solve
	chID, _ := strconv.Atoi(channelID)
	ch, err := dao.FindChannelByID(chID)
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	_sequence, _ := strconv.Atoi(sequence)
	_initReq, _ := strconv.ParseBool(initRequired)

	cc, err := factory.NewChaincodeFactory().
		NewChaincode(nickname, ch.ID, ch.NetworkID, label, policyStr, version, int64(_sequence), _initReq)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	ccSvc := service.NewChaincodeService(cc)

	err = c.SaveUploadedFile(srcTarGz, filepath.Join(cc.GetCCPath(), "src.tar.gz"))
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	go func() {
		chorgs, err := dao.FindAllOrganizationsInChannel(ch)
		if err != nil {
			global.Logger.Error("fail to get orgs", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}
		netorgs, err := dao.FindAllOrganizationsInNetwork(ch.NetworkID)
		if err != nil {
			global.Logger.Error("fail to get all orgs", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}
		orderers, err := dao.FindAllOrderersInNetwork(ch.NetworkID)
		if err != nil {
			global.Logger.Error("fail to get peers", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}

		// 2. unpack
		global.Logger.Info("unpack chaincode")
		if err := ccSvc.Unpack(); err != nil {
			global.Logger.Error("fail to unpack cc", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}

		// 3. build
		global.Logger.Info("build chaincode")
		ccSvc := service.NewChaincodeService(cc)
		dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusBuilding)
		if err := ccSvc.Build(); err != nil {
			global.Logger.Error("fail to build cc ", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}

		// 4. install (any peers)
		global.Logger.Info("install chaincode")
		global.Logger.Info(fmt.Sprintf("%v", ch))
		orgs, err := dao.FindAllOrganizationsInChannel(ch)
		if err != nil {
			global.Logger.Error("fail to get orgs ", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}
		for _, org := range orgs {
			global.Logger.Info("Obtaining rc...")
			adminUser, err := dao.FindSystemUserInOrganization(org.ID)
			if err != nil {
				global.Logger.Error("fail to get adminUser", zap.Error(err))
				continue
			}
			rc, err := sdk.NewSDKClientFactory().NewResmgmtClient(adminUser)
			if err != nil {
				global.Logger.Error("fail to get rc", zap.Error(err))
				continue
			}
			if err := ccSvc.InstallCC(rc); err != nil {
				global.Logger.Error(fmt.Sprintf("fail to install cc to org%d", org.ID), zap.Error(err))
				continue
			}
			break
		}
		if cc.PackageID == "" {
			// 任何一次成功的安装都将更新这个值
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}

		// 5. approve (channel's org)
		global.Logger.Info("approve chaincode")
		for _, org := range chorgs {
			global.Logger.Info(fmt.Sprintf("%s approve cc", org.GetName()))
			adminUser, err := dao.FindSystemUserInOrganization(org.ID)
			if err != nil {
				global.Logger.Error("fail to get adminUser", zap.Error(err))
				dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
				return
			}
			rc, err := sdk.NewSDKClientFactory().NewResmgmtClient(adminUser)
			if err != nil {
				global.Logger.Error("fail to get rc", zap.Error(err))
				dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
				return
			}

			peers, err := dao.FindAllPeersInOrganization(org.ID)
			if err != nil {
				global.Logger.Error("fail to get peers", zap.Error(err))
				dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
				return
			}

			if err := ccSvc.ApproveCC(
				rc,
				orderers[0].GetName(),
				peers[0].GetName()); err != nil {
				global.Logger.Error("fail to approve cc", zap.Error(err))
				dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
				return
			}

			resp, _ := ccSvc.CheckCCCommitReadiness(rc)
			global.Logger.Info(fmt.Sprintf("%v", resp))
		}

		// 6. commmit
		global.Logger.Info("commit chaincode")
		peerURLs := []string{}
		for _, org := range netorgs {
			if org.IsOrdererOrganization() {
				continue
			}
			peers, err := dao.FindAllPeersInOrganization(org.ID)
			if err != nil {
				global.Logger.Error("fail to get peer", zap.Error(err))
				dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
				return
			}
			for _, peer := range peers {
				peerURLs = append(peerURLs, peer.GetName())
			}
		}

		global.Logger.Info("Obtaining rc...")
		adminUser, err := dao.FindSystemUserInOrganization(chorgs[0].ID)
		if err != nil {
			global.Logger.Error("fail to get adminUser", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}
		rc, err := sdk.NewSDKClientFactory().NewResmgmtClient(adminUser)
		if err != nil {
			global.Logger.Error("fail to get rc", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}

		if err := ccSvc.CommitCC(rc, orderers[0].GetName(), peerURLs...); err != nil {
			global.Logger.Error("fail to commit cc", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}

		// 7. start cc container
		if err := ccSvc.CreateEntity(); err != nil {
			global.Logger.Error("fail to create cc container", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}

		dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusRunning)

		global.Logger.Info(fmt.Sprintf("chaincode%d has been created successfully", cc.ID))
	}()

	response.Ok().
		Result(c.JSON)
}

// PATCH /api/chaincode/peer
func InstallChaincode(c *gin.Context)  {
	var info struct{
		Peers 		[]string 	`form:"peers" json:"peers"  binding:"required"`
		ChaincodeID int 		`form:"chaincodeID" json:"chaincodeID" binding:"required"`
	}

	cc, err := dao.FindChaincodeByID(info.ChaincodeID)
	if err != nil {
		global.Logger.Error("fail to get cc", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	ccSvc := service.NewChaincodeService(cc)

	for _, p := range info.Peers {
		global.Logger.Info("Obtaining rc...")
		pCauser := factory.NewCaUserFactory().NewCaUserFromDomainName(p)
		org, err := dao.FindOrganizationByID(pCauser.OrganizationID)
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}

		adminUser, err := dao.FindSystemUserInOrganization(org.ID)
		if err != nil {
			global.Logger.Error("fail to get adminUser", zap.Error(err))
			response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
		rc, err := sdk.NewSDKClientFactory().NewResmgmtClient(adminUser)
		if err != nil {
			global.Logger.Error("fail to get rc", zap.Error(err))
			response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}

		if err := ccSvc.InstallCC(rc, p); err != nil {
			global.Logger.Error(fmt.Sprintf("fail to install cc to org%d", org.ID), zap.Error(err))
			response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
	}
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

	cc, err := dao.FindChaincodeByID(ccID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	ccSvc := service.NewChaincodeService(cc)

	ch, err := dao.FindChannelByID(cc.ChannelID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	orgs, err := dao.FindAllOrganizationsInChannel(ch)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	global.Logger.Info("Obtaining channel client...")
	adminUser, err := dao.FindSystemUserInOrganization(orgs[0].ID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	chClient, err := sdk.NewSDKClientFactory().NewChannelClient(adminUser, ch)
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
		ret, err = ccSvc.InitCC(chClient, info.Args, info.PeerURLs...)
	case "query":
		ret, err = ccSvc.QueryCC(chClient, info.Args, info.PeerURLs...)
	case "execute":
		ret, err = ccSvc.ExecuteCC(chClient, info.Args, info.PeerURLs...)
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

// GET /api/chaincode
// ChaincodeInstance
func ListChaincodes(c *gin.Context) {
	ccs, err := dao.FindAllChaincodes()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().
		SetPayload(response.NewChaincodes(ccs)).
		Result(c.JSON)
}