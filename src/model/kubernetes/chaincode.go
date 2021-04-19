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

type Chaincode struct {
	callback
	ChaincodeID		int
	PackageID		string
	ChannelID 		int
	NetworkID 		int
}

func NewChaincode(netID int, channelID int, packageID string, chaincodeID int) *Chaincode {
	return &Chaincode{
		NetworkID: netID,
		ChannelID: channelID,
		PackageID: packageID,
		ChaincodeID: chaincodeID,
	}
}

func (cc *Chaincode) GetName() string {
	return fmt.Sprintf(
		"cc%d-chan%d-net%d",
		cc.ChaincodeID,
		cc.ChannelID,
		cc.NetworkID)
}

func (cc *Chaincode) GetSelector() map[string]string {
	return map[string]string{
		"app": "mictract",
		"net": strconv.Itoa(cc.NetworkID),
		"channel": strconv.Itoa(cc.ChannelID),
		"chaincode": strconv.Itoa(cc.ChaincodeID),
		"tier": "chaincode",
	}
}

func (cc *Chaincode) GetPod() (*apiv1.Pod, error) {
	return getPod(cc)
}

// Connect to K8S to create the configMap.
func (cc *Chaincode) CreateConfigMap() {
	name := cc.GetName()

	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "-env",
		},
		Data:	map[string]string{
			"WHISPER": "Marx bless, no bugs",
			"CHAINCODE_ADDRESS": "0.0.0.0:9999",
			"CHAINCODE_CCID": cc.PackageID,
		},
	}

	_, err := global.K8sClientset.CoreV1().
		ConfigMaps(apiv1.NamespaceDefault).
		Create(context.TODO(), configMap, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create chaincode config map error", zap.Error(err))
	}
}

// Connect to K8S to create the deployment.
func (cc *Chaincode) CreateDeployment() {
	name := cc.GetName()

	matchlabels := map[string]string{
		"app": "mictract",
		"net": strconv.Itoa(cc.NetworkID),
		"channel": strconv.Itoa(cc.ChannelID),
		"chaincode": strconv.Itoa(cc.ChaincodeID),
		"tier": "chaincode",
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
							Name:  "chaincode",
							Image: "alpine:3.11",
							Command: []string{"/host/var/run/chaincode"},
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
									Name:          "chaincode",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 9999,
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:             "cc",
									MountPath:        "/host/var/run/chaincode",
									SubPath: filepath.Join(
										"chaincodes",
										fmt.Sprintf("chaincode%d", cc.ChaincodeID),
										"chaincode"),
								},
							},
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name:         "cc",
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
		global.Logger.Error("Create chaincode deployment error", zap.Error(err))
	}
}

// Connect to K8S to create the service.
func (cc *Chaincode) CreateService() {
	name := cc.GetName()

	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiv1.ServiceSpec{
			Type: apiv1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": "mictract",
				"net": strconv.Itoa(cc.NetworkID),
				"channel": strconv.Itoa(cc.ChannelID),
				"chaincode": strconv.Itoa(cc.ChaincodeID),
				"tier": "chaincode",
			},
			Ports: []apiv1.ServicePort{
				{
					Name: "chaincode",
					Port: 9999,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "chaincode",
					},
				},
			},
		},
	}

	_, err := global.K8sClientset.CoreV1().
		Services(apiv1.NamespaceDefault).
		Create(context.TODO(), service, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create chaincode service error", zap.Error(err))
	}
}

// Connect to K8S to create all the resources.
func (cc *Chaincode) Create() {
	cc.CreateConfigMap()
	cc.CreateDeployment()
	cc.CreateService()
}

func (cc *Chaincode) AwaitableCreate() error {
	return awaitableCreate(cc)
}

// Connect to K8S to delete all the resources.
func (cc *Chaincode) Delete() {
	var err error
	name := cc.GetName()

	err = global.K8sClientset.CoreV1().
		ConfigMaps(apiv1.NamespaceDefault).
		Delete(context.TODO(), name + "-env", metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete chaincode config map error", zap.Error(err))
	}

	err = global.K8sClientset.AppsV1().
		Deployments(apiv1.NamespaceDefault).
		Delete(context.TODO(), name, metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete chaincode deployment error", zap.Error(err))
	}

	err = global.K8sClientset.CoreV1().
		Services(apiv1.NamespaceDefault).
		Delete(context.TODO(), name, metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete chaincode service error", zap.Error(err))
	}
}

func (cc *Chaincode) Watch() {
	watch(cc, &cc.callback)
}

func (cc *Chaincode) ExecCommand(cmd ...string) (string, string, error) {
	return execCommand(cc, cmd...)
}
