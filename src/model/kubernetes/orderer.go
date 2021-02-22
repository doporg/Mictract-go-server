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

type Orderer struct {
	callback
	OrdererID 	int
	NetworkID 	int
}

func NewOrderer(netID int, ordererID int) *Orderer {
	return &Orderer{NetworkID: netID, OrdererID: ordererID}
}

// Get orderer name.
// Example: orderer1-net1
func (o *Orderer) GetName() string {
	return fmt.Sprintf("orderer%d-net%d", o.OrdererID, o.NetworkID)
}

// Get orderer domain.
// Example: orderer1.net1.com
func (o *Orderer) GetDomain() string {
	return fmt.Sprintf("orderer%d.net%d.com", o.OrdererID, o.NetworkID)
}

// Get orderer sub path.
// Example: networks/net1/ordererOrganizations/net1.com/orderers/orderer1.net1.com/
func (o *Orderer) GetSubPath() string {
	netDomain := fmt.Sprintf("net%d.com", o.NetworkID)
	ordererDomain := o.GetDomain()

	return filepath.Join("networks", "net" + strconv.Itoa(o.NetworkID),
		"ordererOrganizations", netDomain,
		"orderers", ordererDomain,
	)
}

func (o *Orderer) GetSelector() map[string]string {
	return map[string]string{
		"app": "mictract",
		"net": strconv.Itoa(o.NetworkID),
		"orderer": strconv.Itoa(o.OrdererID),
		"tier": "orderer",
	}
}

func (o *Orderer) GetPod() (*apiv1.Pod, error) {
	return getPod(o)
}

// Connect to K8S to create the configMap.
func (o *Orderer) CreateConfigMap() {
	name := o.GetName()
	// Note: local MSP id should be like "Orderer1MSP", which is written in `fabric-org1-config.yaml` and can not be changed.
	localMSPId := fmt.Sprintf("Orderer%dMSP", o.OrdererID)

	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "-env",
		},
		Data:	map[string]string{
			"FABRIC_LOGGING_SPEC":"INFO",
			"ORDERER_GENERAL_LISTENADDRESS":"0.0.0.0",
			"ORDERER_GENERAL_LISTENPORT":"7050",
			"ORDERER_GENERAL_GENESISMETHOD":"file",
			"ORDERER_GENERAL_GENESISFILE":"/var/hyperledger/orderer/orderer.genesis.block",
			"ORDERER_GENERAL_LOCALMSPID":localMSPId,
			"ORDERER_GENERAL_LOCALMSPDIR":"/var/hyperledger/orderer/msp",
			"ORDERER_GENERAL_TLS_ENABLED":"true",
			"ORDERER_GENERAL_TLS_PRIVATEKEY":"/var/hyperledger/orderer/tls/server.key",
			"ORDERER_GENERAL_TLS_CERTIFICATE":"/var/hyperledger/orderer/tls/server.crt",
			"ORDERER_GENERAL_TLS_ROOTCAS":"[/var/hyperledger/orderer/tls/ca.crt]",
			"ORDERER_KAFKA_TOPIC_REPLICATIONFACTOR":"1",
			"ORDERER_KAFKA_VERBOSE":"true",
			"ORDERER_GENERAL_CLUSTER_CLIENTCERTIFICATE":"/var/hyperledger/orderer/tls/server.crt",
			"ORDERER_GENERAL_CLUSTER_CLIENTPRIVATEKEY":"/var/hyperledger/orderer/tls/server.key",
			"ORDERER_GENERAL_CLUSTER_ROOTCAS":"[/var/hyperledger/orderer/tls/ca.crt]",
		},
	}

	_, err := global.K8sClientset.CoreV1().
		ConfigMaps(apiv1.NamespaceDefault).
		Create(context.TODO(), configMap, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create orderer config map error", zap.Error(err))
	}
}

// Connect to K8S to create the deployment.
func (o *Orderer) CreateDeployment() {
	netPath := filepath.Join("network", "net" + strconv.Itoa(o.NetworkID))
	subPath := o.GetSubPath()
	name := o.GetName()

	matchLabels := map[string]string{
		"app": "mictract",
		"net": strconv.Itoa(o.NetworkID),
		"orderer": strconv.Itoa(o.OrdererID),
		"tier": "orderer",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: matchLabels,
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "orderer",
							Image: "hyperledger/fabric-orderer:2.2.1",
							Command: []string{ "orderer" },
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
									Name:          "orderer",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 7050,
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:             "genesis-block",
									MountPath:        "/var/hyperledger/orderer/orderer.genesis.block",
									SubPath: filepath.Join(netPath, "genesis.block"),
								},
								{
									Name:             "msp",
									MountPath:        "/etc/hyperledger/orderer/msp",
									SubPath: filepath.Join(subPath, "msp"),
								},
								{
									Name:             "tls",
									MountPath:        "/etc/hyperledger/orderer/tls",
									SubPath: filepath.Join(subPath, "tls"),
								},
								{
									Name:             "production",
									MountPath:        "/var/hyperledger/production/orderer",
									SubPath: filepath.Join(subPath, "prod"),
								},
							},
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name:          "genesis-block",
							VolumeSource: apiv1.VolumeSource{
								NFS: &apiv1.NFSVolumeSource{
									Server: config.NFS_SERVER_URL,
									Path: 	  config.NFS_EXPOSED_PATH,
								},
							},
						},
						{
							Name:         "msp",
							VolumeSource: apiv1.VolumeSource{
								NFS: &apiv1.NFSVolumeSource{
									Server: config.NFS_SERVER_URL,
									Path: 	  config.NFS_EXPOSED_PATH,
								},
							},
						},
						{
							Name:         "tls",
							VolumeSource: apiv1.VolumeSource{
								NFS: &apiv1.NFSVolumeSource{
									Server: config.NFS_SERVER_URL,
									Path: 	  config.NFS_EXPOSED_PATH,
								},
							},
						},
						{
							Name:         "production",
							VolumeSource: apiv1.VolumeSource{
								NFS: &apiv1.NFSVolumeSource{
									Server: config.NFS_SERVER_URL,
									Path:     config.NFS_EXPOSED_PATH,
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
func (o *Orderer) CreateService() {
	netID := strconv.Itoa(o.NetworkID)
	ordererID := strconv.Itoa(o.OrdererID)
	name := o.GetName()

	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiv1.ServiceSpec{
			Type: apiv1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": "mictract",
				"net": netID,
				"orderer": ordererID,
				"tier": "orderer",
			},
			Ports: []apiv1.ServicePort{
				{
					Name: "orderer",
					Port: 7050,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "orderer",
					},
				},
			},
		},
	}

	_, err := global.K8sClientset.CoreV1().
		Services(apiv1.NamespaceDefault).
		Create(context.TODO(), service, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create orderer service error", zap.Error(err))
	}
}

// Connect to K8S to create all the resources.
func (o *Orderer) Create() {
	o.CreateConfigMap()
	o.CreateDeployment()
	o.CreateService()
}

// Connect to K8S to delete all the resources.
func (o *Orderer) Delete() {
	var err error
	name := o.GetName()

	err = global.K8sClientset.CoreV1().
		ConfigMaps(apiv1.NamespaceDefault).
		Delete(context.TODO(), name + "-env", metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete orderer config map error", zap.Error(err))
	}

	err = global.K8sClientset.AppsV1().
		Deployments(apiv1.NamespaceDefault).
		Delete(context.TODO(), name, metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete orderer deployment error", zap.Error(err))
	}

	err = global.K8sClientset.CoreV1().
		Services(apiv1.NamespaceDefault).
		Delete(context.TODO(), name, metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete orderer service error", zap.Error(err))
	}
}

func (o *Orderer) Watch() {
	watch(o, &o.callback)
}

func (o *Orderer) ExecCommand(cmd ...string) (string, string, error) {
	return execCommand(o, cmd...)
}
