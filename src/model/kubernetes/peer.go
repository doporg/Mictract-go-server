package kubernetes

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"mictract/config"
	"mictract/global"
	"path/filepath"
	"strconv"
)

type Peer struct {
	callback
	PeerID			int
	OrganizationID	int
	NetworkID 		int
}

func NewPeer(netID int, orgID int, peerID int) *Peer {
	return &Peer{NetworkID: netID, OrganizationID: orgID, PeerID: peerID}
}

// Get peer name.
// Example: peer1-org1-net1
func (p *Peer) GetName() string {
	return fmt.Sprintf("peer%d-org%d-net%d", p.PeerID, p.OrganizationID, p.NetworkID)
}

// Get peer domain.
// Example: peer1.org1.net1.com
func (p *Peer) GetDomain() string {
	return fmt.Sprintf("peer%d.org%d.net%d.com", p.PeerID, p.OrganizationID, p.NetworkID)
}

// Get peer sub path.
// Example: networks/net1/peerOrganizations/org1.net1.com/peers/peer1.org1.net1.com/
func (p *Peer) GetSubPath() string {
	orgDomain := fmt.Sprintf("org%d.net%d.com", p.OrganizationID, p.NetworkID)
	peerDomain := p.GetDomain()

	return filepath.Join("networks", "net" + strconv.Itoa(p.NetworkID),
		"peerOrganizations", orgDomain,
		"peers", peerDomain,
	)
}

func (p *Peer) GetSelector() map[string]string {
	return map[string]string{
		"app": "mictract",
		"net": strconv.Itoa(p.NetworkID),
		"org": strconv.Itoa(p.OrganizationID),
		"peer": strconv.Itoa(p.PeerID),
		"tier": "peer",
	}
}

func (p *Peer) GetPod() (*apiv1.Pod, error) {
	return getPod(p)
}

// Connect to K8S to create the configMap.
func (p *Peer) CreateConfigMap() {
	name := p.GetName()
	peerId := p.GetName()
	// Note: local MSP id should be like "Org1MSP", which is written in `fabric-org1-config.yaml` and can not be changed.
	localMSPId := fmt.Sprintf("Org%dMSP", p.OrganizationID)

	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "-env",
		},
		Data:	map[string]string{
			// These two env args are to indicate the communication address between the peer and the docker, when deploy a chaincode
			"CORE_VM_ENDPOINT":"unix:///host/var/run/docker.sock",
			"CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE":"test",

			"FABRIC_LOGGING_SPEC":"INFO",
			"CORE_PEER_TLS_ENABLED":"true",
			"CORE_PEER_PROFILE_ENABLED":"true",
			"CORE_PEER_TLS_CERT_FILE":"/etc/hyperledger/fabric/tls/server.crt",
			"CORE_PEER_TLS_KEY_FILE":"/etc/hyperledger/fabric/tls/server.key",
			"CORE_PEER_TLS_ROOTCERT_FILE":"/etc/hyperledger/fabric/tls/ca.crt",
			"CORE_PEER_ID":peerId,
			"CORE_PEER_ADDRESS":peerId + ":7051",
			"CORE_PEER_LISTENADDRESS":"0.0.0.0:7051",

			// The address to connect chaincode
			"CORE_PEER_CHAINCODEADDRESS":peerId + ":7052",
			"CORE_PEER_CHAINCODELISTENADDRESS":"0.0.0.0:7052",

			"CORE_PEER_GOSSIP_BOOTSTRAP":peerId + ":7051",
			"CORE_PEER_GOSSIP_EXTERNALENDPOINT":peerId + ":7051",
			"CORE_PEER_LOCALMSPID": localMSPId,
		},
	}

	_, err := global.K8sClientset.CoreV1().
		ConfigMaps(apiv1.NamespaceDefault).
		Create(context.TODO(), configMap, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create peer config map error", zap.Error(err))
	}
}

// Connect to K8S to create the deployment.
func (p *Peer) CreateDeployment() {
	subPath := p.GetSubPath()
	name := p.GetName()

	matchlabels := map[string]string{
		"app": "mictract",
		"net": strconv.Itoa(p.NetworkID),
		"org": strconv.Itoa(p.OrganizationID),
		"peer": strconv.Itoa(p.PeerID),
		"tier": "peer",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchlabels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: matchlabels,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "peer",
							Image: "hyperledger/fabric-peer:2.2.1",
							Command: []string{ "sh", "-c",
								"GOPROXY=https://goproxy.io,direct peer node start --peer-chaincodedev" },
							EnvFrom: []apiv1.EnvFromSource{
								{
									ConfigMapRef: &apiv1.ConfigMapEnvSource{
										LocalObjectReference: apiv1.LocalObjectReference{
											Name: name + "-env",
										},
									},
								},
							},
							Ports: []apiv1.ContainerPort{
								{
									Name:          "peer",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 7051,
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:             "host",
									MountPath:        "/host/var/run",
									SubPath: filepath.Join(subPath, "run"),
								},
								{
									Name:             "msp",
									MountPath:        "/etc/hyperledger/fabric/msp",
									SubPath: filepath.Join(subPath, "msp"),
								},
								{
									Name:             "tls",
									MountPath:        "/etc/hyperledger/fabric/tls",
									SubPath: filepath.Join(subPath, "tls"),
								},
								{
									Name:             "production",
									MountPath:        "/var/hyperledger/production",
									SubPath: filepath.Join(subPath, "prod"),
								},
							},
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name:         "host",
							VolumeSource: apiv1.VolumeSource{
								NFS: &apiv1.NFSVolumeSource{
									Server: config.NFS_SERVER_URL,
									Path: config.NFS_EXPOSED_PATH,
								},
							},
						},
						{
							Name:         "msp",
							VolumeSource: apiv1.VolumeSource{
								NFS: &apiv1.NFSVolumeSource{
									Server: config.NFS_SERVER_URL,
									Path: config.NFS_EXPOSED_PATH,
								},
							},
						},
						{
							Name:         "tls",
							VolumeSource: apiv1.VolumeSource{
								NFS: &apiv1.NFSVolumeSource{
									Server: config.NFS_SERVER_URL,
									Path: config.NFS_EXPOSED_PATH,
								},
							},
						},
						{
							Name:         "production",
							VolumeSource: apiv1.VolumeSource{
								NFS: &apiv1.NFSVolumeSource{
									Server: config.NFS_SERVER_URL,
									Path: config.NFS_EXPOSED_PATH,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := global.K8sClientset.AppsV1().
		Deployments(apiv1.NamespaceDefault).
		Create(context.TODO(), deployment, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create peer deployment error", zap.Error(err))
	}
}

// Connect to K8S to create the service.
func (p *Peer) CreateService() {
	netID := strconv.Itoa(p.NetworkID)
	orgID := strconv.Itoa(p.OrganizationID)
	peerID := strconv.Itoa(p.PeerID)
	name := p.GetName()

	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiv1.ServiceSpec{
			Type: apiv1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": "mictract",
				"net": netID,
				"org": orgID,
				"peer": peerID,
				"tier": "peer",
			},
			Ports: []apiv1.ServicePort{
				{
					Name: "peer",
					Port: 7051,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "peer",
					},
				},
			},
		},
	}

	_, err := global.K8sClientset.CoreV1().
		Services(apiv1.NamespaceDefault).
		Create(context.TODO(), service, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create peer service error", zap.Error(err))
	}
}

// Connect to K8S to create all the resources.
func (p *Peer) Create() {
	p.CreateConfigMap()
	p.CreateDeployment()
	p.CreateService()
}

// Connect to K8S to delete all the resources.
func (p *Peer) Delete() {
	var err error
	name := p.GetName()

	err = global.K8sClientset.CoreV1().
		ConfigMaps(apiv1.NamespaceDefault).
		Delete(context.TODO(), name + "-env", metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete peer config map error", zap.Error(err))
	}

	err = global.K8sClientset.AppsV1().
		Deployments(apiv1.NamespaceDefault).
		Delete(context.TODO(), name, metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete peer deployment error", zap.Error(err))
	}

	err = global.K8sClientset.CoreV1().
		Services(apiv1.NamespaceDefault).
		Delete(context.TODO(), name, metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete peer service error", zap.Error(err))
	}
}

func (p *Peer) Watch() {
	watch(p, &p.callback)
}

func (p *Peer) ExecCommand(cmd ...string) (string, string, error) {
	return execCommand(p, cmd...)
}
