package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"mictract/dao"
	"mictract/enum"
	"mictract/global"
	"mictract/model"
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

	go func(ccSvc service.ChaincodeService, ch model.Channel) {
		chorgs, err := dao.FindAllOrganizationsInChannel(&ch)
		if err != nil {
			global.Logger.Error("fail to get orgs", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}
		orderers, err := dao.FindAllOrderersInNetwork(ch.NetworkID)
		if err != nil {
			global.Logger.Error("fail to get peers", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}
		peers, err := dao.FindAllPeersInChannel(&ch)
		if err != nil {
			global.Logger.Error("fail to get peer", zap.Error(err))
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

		// install all peer in channel
		go func() {
			orgs, _ := dao.FindAllOrganizationsInChannel(&ch)
			for _, org := range orgs {
				go func(org model.Organization) {
					global.Logger.Info(fmt.Sprintf("install %s to %s", cc.GetName(), org.GetName()))
					adminUser, err := dao.FindSystemUserInOrganization(org.ID)
					if err != nil {
						return
					}
					rc, err := sdk.NewSDKClientFactory().NewResmgmtClient(adminUser)
					if err != nil {
						return
					}
					if err := ccSvc.InstallCC(rc); err != nil {
						return
					}
				}(org)
			}
		}()

		// 3. build
		global.Logger.Info("build chaincode")
		dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusBuilding)
		if err := ccSvc.Build(); err != nil {
			global.Logger.Error("fail to build cc ", zap.Error(err))
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

			retry_count := 3
			for retry_count >= 0 {
				if err := ccSvc.ApproveCC(
					rc,
					orderers[0].GetName(),
					peers[0].GetName()); err != nil {
					global.Logger.Error("fail to approve cc", zap.Error(err))
					if retry_count <= 0 {
						dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
						return
					}
					global.Logger.Info("Retrying")
					retry_count --
				} else {
					break
				}
			}

			resp, _ := ccSvc.CheckCCCommitReadiness(rc)
			global.Logger.Info(fmt.Sprintf("%v", resp))
		}

		// 6. commmit
		// note: "implicit policy evaluation failed" if use rc(include org)
		//       you should use rc(include network) to get enough endoerment
		global.Logger.Info("commit chaincode")
		orgID := chorgs[0].ID
		peerURLs := []string{}
		for _, peer := range peers {
			peerURLs = append(peerURLs, peer.GetName())
		}

		global.Logger.Info("Obtaining rc(include network)...")
		adminUser, err := dao.FindSystemUserInOrganization(orgID)
		if err != nil {
			global.Logger.Error("fail to get adminUser", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}
		rc, err := sdk.NewSDKClientFactory().NewResmgmtClientIncludeNetwork(adminUser)
		if err != nil {
			global.Logger.Error("fail to get rc", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}

		retry_count := 3
		for retry_count >= 0 {
			if err := ccSvc.CommitCC(
				rc,
				orderers[0].GetName(),
				peerURLs...,
			); err != nil {
				global.Logger.Error("fail to commit cc", zap.Error(err))
				if retry_count == 0{
					dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
					return
				}
				global.Logger.Error("Retrying")
				retry_count --
			} else {
				break
			}
		}

		// 7. start cc container
		global.Logger.Info("start cc container")
		if err := ccSvc.CreateEntity(); err != nil {
			global.Logger.Error("fail to create cc container", zap.Error(err))
			dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
			return
		}

		dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusRunning)

		global.Logger.Info(fmt.Sprintf("chaincode%d has been created successfully", cc.ID))
	}(*ccSvc, *ch)

	response.Ok().
		Result(c.JSON)
}

// POST /api/chaincode/install
func InstallChaincode(c *gin.Context)  {
	var info struct{
		Peers 		[]string 	`form:"peers" json:"peers"  binding:"required"`
		ChaincodeID int 		`form:"id" json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
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
	response.Ok().Result(c.JSON)
}

// POST /api/chaincode/approve
func ApproveChaincode(c *gin.Context)  {
	var info struct{
		OrganizationID 	int 	`form:"organizationID" json:"organizationID" binding:"required"`
		ChaincodeID 	int 	`form:"id" json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	cc, err := dao.FindChaincodeByID(info.ChaincodeID)
	if err != nil {
		global.Logger.Error("fail to get cc", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	ccSvc := service.NewChaincodeService(cc)

	adminUser, err := dao.FindSystemUserInOrganization(info.OrganizationID)
	if err != nil {
		global.Logger.Error("fail to get adminUser", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	rc, err := sdk.NewSDKClientFactory().NewResmgmtClient(adminUser)
	if err != nil {
		global.Logger.Error("fail to get rc", zap.Error(err))
		dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
		return
	}

	peers, err := dao.FindAllPeersInOrganization(info.OrganizationID)
	if err != nil {
		global.Logger.Error("fail to get peers", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	orderers, err := dao.FindAllOrderersInNetwork(adminUser.NetworkID)
	if err != nil {
		global.Logger.Error("fail to get orderers", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := ccSvc.ApproveCC(
		rc,
		orderers[0].GetName(),
		peers[0].GetName()); err != nil {
		global.Logger.Error("fail to approve cc", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrBlockchainNetworkError).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	resp, _ := ccSvc.CheckCCCommitReadiness(rc)
	global.Logger.Info(fmt.Sprintf("%v", resp))

	response.Ok().Result(c.JSON)
}

// POST /api/chaincode/commit
func CommitChaincode(c *gin.Context)  {
	var info struct{
		ChaincodeID 	int 	`form:"id" json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	cc, err := dao.FindChaincodeByID(info.ChaincodeID)
	if err != nil {
		global.Logger.Error("fail to get cc", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	ccSvc := service.NewChaincodeService(cc)

	ch, err := dao.FindChannelByID(cc.ChannelID)
	if err != nil {
		global.Logger.Error("fail to get channel", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	orgs, err := dao.FindAllOrganizationsInChannel(ch)
	if err != nil {
		global.Logger.Error("fail to get orgs", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	peers, err := dao.FindAllPeersInChannel(ch)
	if err != nil {
		global.Logger.Error("fail to get peers", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	orderers, err := dao.FindAllOrderersInNetwork(ch.NetworkID)
	if err != nil {
		global.Logger.Error("fail to get orderers", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}



	orgID := orgs[0].ID
	peerURLs := []string{}
	for _, peer := range peers {
		peerURLs = append(peerURLs, peer.GetName())
	}

	global.Logger.Info("Obtaining rc(include network)...")
	adminUser, err := dao.FindSystemUserInOrganization(orgID)
	if err != nil {
		global.Logger.Error("fail to get adminUser", zap.Error(err))
		dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusError)
		return
	}
	rc, err := sdk.NewSDKClientFactory().NewResmgmtClientIncludeNetwork(adminUser)
	if err != nil {
		global.Logger.Error("fail to get rc", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	if err := ccSvc.CommitCC(
		rc,
		orderers[0].GetName(),
		peerURLs...,
	); err != nil {
		global.Logger.Error("fail to commit cc", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	response.Ok().Result(c.JSON)
}

// POST /api/chaincode/start
func StartChaincodeEntity(c *gin.Context)  {
	var info struct{
		ChaincodeID 	int 	`form:"id" json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}
	cc, err := dao.FindChaincodeByID(info.ChaincodeID)
	if err != nil {
		global.Logger.Error("fail to get cc", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	ccSvc := service.NewChaincodeService(cc)
	if err := ccSvc.CreateEntity(); err != nil {
		global.Logger.Error("fail to create cc container", zap.Error(err))
		response.Err(http.StatusInternalServerError, enum.CodeErrDB).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	dao.UpdateChaincodeStatusByID(cc.ID, enum.StatusRunning)
	response.Ok().Result(c.JSON)
}

// GET /api/chaincode
func ListChaincodes(c *gin.Context) {
	info := struct {
		NetworkID int `form:"networkID"`
	}{}

	if err := c.ShouldBindQuery(&info); err != nil {
		response.Err(http.StatusBadRequest, enum.CodeErrMissingArgument).
			SetMessage(err.Error()).
			Result(c.JSON)
		return
	}

	var ccs []model.Chaincode
	var err error

	if info.NetworkID == 0 {
		ccs, err = dao.FindAllChaincodes()
	} else {
		ccs, err = dao.FindAllChaincodesInNetwork(info.NetworkID)
	}
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