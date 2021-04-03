package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	channelclient "github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"go.uber.org/zap"
	"mictract/enum"
	"mictract/global"
	"mictract/model"
	"mictract/model/request"
	"mictract/model/response"
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
	ch, err := model.FindChannelByID(chID)
	if err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	_sequence, _ := strconv.Atoi(sequence)
	_initReq, _ := strconv.ParseBool(initRequired)

	cc, err := model.NewChaincode(nickname, ch.ID, ch.NetworkID, label, policyStr, version, int64(_sequence), _initReq)
	if err != nil {
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
		chorgs, err := ch.GetOrganizations()
		if err != nil {
			global.Logger.Error("fail to get orgs", zap.Error(err))
			cc.UpdateStatus(enum.StatusError)
			return
		}
		net, err := model.FindNetworkByID(ch.NetworkID)
		if err != nil {
			global.Logger.Error("fail to get net", zap.Error(err))
			cc.UpdateStatus(enum.StatusError)
			return
		}
		netorgs, err := net.GetOrganizations()
		if err != nil {
			global.Logger.Error("fail to get all orgs", zap.Error(err))
			cc.UpdateStatus(enum.StatusError)
			return
		}
		orderers, err := net.GetOrderers()
		if err != nil {
			global.Logger.Error("fail to get peers", zap.Error(err))
			cc.UpdateStatus(enum.StatusError)
			return
		}

		// 2. unpack
		if err := cc.Unpack(); err != nil {
			global.Logger.Error("fail to unpack cc", zap.Error(err))
			cc.UpdateStatus(enum.StatusError)
			return
		}

		// 3. build
		cc.UpdateStatus(enum.StatusBuilding)
		if err := cc.Build(); err != nil {
			global.Logger.Error("fail to build cc ", zap.Error(err))
			cc.UpdateStatus(enum.StatusError)
			return
		}

		// 4. install (all peers)
		global.Logger.Info(fmt.Sprintf("%v", ch))
		orgs, err := ch.GetOrganizations()
		if err != nil {
			global.Logger.Error("fail to get orgs ", zap.Error(err))
			cc.UpdateStatus(enum.StatusError)
			return
		}
		for _, org := range orgs {
			global.Logger.Info("Obtaining rc...")
			adminUser, err := org.GetSystemUser()
			if err != nil {
				global.Logger.Error("fail to get adminUser", zap.Error(err))
				//cc.UpdateStatus(enum.StatusError)
				//return
			}
			rc, err := ch.NewResmgmtClient(adminUser.GetName(), org.GetName())
			if err != nil {
				global.Logger.Error("fail to get rc", zap.Error(err))
				//cc.UpdateStatus(enum.StatusError)
				//return
			}

			if err := cc.InstallCC(rc); err != nil {
				global.Logger.Error(fmt.Sprintf("fail to install cc to org%d", org.ID), zap.Error(err))
				// cc.UpdateStatus(enum.StatusError)
				// return
			}
		}
		if cc.PackageID == "" {
			// 任何一次成功的安装都将更新这个值
			cc.UpdateStatus(enum.StatusError)
			return
		}

		// 5. approve (channel's org)
		global.Logger.Info("Obtaining sdk...")

		for _, org := range chorgs {
			global.Logger.Info("Obtaining rc...")
			adminUser, err := org.GetSystemUser()
			if err != nil {
				global.Logger.Error("fail to get adminUser", zap.Error(err))
				cc.UpdateStatus(enum.StatusError)
				return
			}
			rc, err := ch.NewResmgmtClient(adminUser.GetName(), org.GetName())
			if err != nil {
				global.Logger.Error("fail to get rc", zap.Error(err))
				cc.UpdateStatus(enum.StatusError)
				return
			}

			peers, err := org.GetPeers()
			if err != nil {
				global.Logger.Error("fail to get peers", zap.Error(err))
				cc.UpdateStatus(enum.StatusError)
				return
			}

			if err := cc.ApproveCC(rc, orderers[0].GetName(), peers[0].GetName()); err != nil {
				global.Logger.Error("fail to get approve cc", zap.Error(err))
				cc.UpdateStatus(enum.StatusError)
				return
			}

			resp, _ := cc.CheckCCCommitReadiness(rc)
			global.Logger.Info(fmt.Sprintf("%v", resp))
		}

		// 6. commmit
		peerURLs := []string{}
		for _, org := range netorgs {
			if org.IsOrdererOrganization() {
				continue
			}
			peers, err := org.GetPeers()
			if err != nil {
				global.Logger.Error("fail to get peer", zap.Error(err))
				cc.UpdateStatus(enum.StatusError)
				return
			}
			for _, peer := range peers {
				peerURLs = append(peerURLs, peer.GetName())
			}
		}

		global.Logger.Info("Obtaining rc...")
		adminUser, err := chorgs[0].GetSystemUser()
		if err != nil {
			global.Logger.Error("fail to get adminUser", zap.Error(err))
			cc.UpdateStatus(enum.StatusError)
			return
		}
		rc, err := ch.NewResmgmtClient(adminUser.GetName(), chorgs[0].GetName())
		if err != nil {
			global.Logger.Error("fail to get rc", zap.Error(err))
			cc.UpdateStatus(enum.StatusError)
			return
		}

		if err := cc.CommitCC(rc, orderers[0].GetName(), peerURLs...); err != nil {
			global.Logger.Error("fail to commit cc", zap.Error(err))
			cc.UpdateStatus(enum.StatusError)
			return
		}

		// 7. start cc container
		if err := cc.CreateEntity(); err != nil {
			global.Logger.Error("fail to create cc container", zap.Error(err))
			cc.UpdateStatus(enum.StatusError)
			return
		}

		cc.UpdateStatus(enum.StatusRunning)

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

	cc, err := model.FindChaincodeByID(info.ChaincodeID)
	if err != nil {
		global.Logger.Error("fail to get cc", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	ch, err := model.FindChannelByID(cc.ChannelID)
	if err != nil {
		global.Logger.Error("fail to get ch", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	for _, p := range info.Peers {
		global.Logger.Info("Obtaining rc...")
		pCauser := model.NewCaUserFromDomainName(p)
		org, err := model.FindOrganizationByID(pCauser.OrganizationID)
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}

		adminUser, err := org.GetSystemUser()
		if err != nil {
			global.Logger.Error("fail to get adminUser", zap.Error(err))
			response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
		rc, err := ch.NewResmgmtClient(adminUser.GetName(), org.GetName())
		if err != nil {
			global.Logger.Error("fail to get rc", zap.Error(err))
			response.Err(http.StatusInternalServerError, enum.CodeErrNotFound).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}

		if err := cc.InstallCC(rc, p); err != nil {
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

	cc, err := model.FindChaincodeByID(ccID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	ch, err := model.FindChannelByID(cc.ChannelID)
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	orgs, err := ch.GetOrganizations()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	global.Logger.Info("Obtaining channel client...")
	adminUser, err := orgs[0].GetSystemUser()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	chClient, err := ch.NewChannelClient(adminUser.GetName(), orgs[0].GetName())
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
		ret, err = cc.InitCC(chClient, info.Args, info.PeerURLs...)
	case "query":
		ret, err = cc.QueryCC(chClient, info.Args, info.PeerURLs...)
	case "execute":
		ret, err = cc.ExecuteCC(chClient, info.Args, info.PeerURLs...)
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
	ccs := []model.Chaincode{}
	nets, err := model.FindAllNetworks()
	if err != nil {
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	for _, net := range nets {
		_ccs, err := net.GetChaincodes()
		if err != nil {
			response.Err(http.StatusInternalServerError, enum.CodeErrDB).
				SetMessage(err.Error()).
				Result(c.JSON)
			return
		}
		ccs = append(ccs, _ccs...)
	}

	response.Ok().
		SetPayload(response.NewChaincodes(ccs)).
		Result(c.JSON)
}