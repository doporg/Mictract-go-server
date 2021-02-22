package kubernetes

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"mictract/config"
	"mictract/global"
	"path/filepath"
	"strconv"
)

type CA struct {
	// Note: OrganizationID < 0  means it is a orderer CA.
	callback
	OrganizationID	int
	NetworkID 		int
}

func NewPeerCA(netID int, orgID int) *CA {
	return &CA{OrganizationID: orgID, NetworkID: netID}
}

func NewOrdererCA(netID int) *CA {
	return &CA{OrganizationID: -1, NetworkID: netID}
}

func (ca *CA) IsOrdererCA() bool {
	return ca.OrganizationID < 0
}

// Get ca name.
// Example: ca-org1-net1
// Example: ca-net1
func (ca *CA) GetName() string {
	if ca.IsOrdererCA() {
		return fmt.Sprintf("ca-net%d", ca.NetworkID)
	}
	return fmt.Sprintf("ca-org%d-net%d", ca.OrganizationID, ca.NetworkID)
}

// Get ca sub path.
// Example: networks/net1/peerOrganizations/org1.net1.com/ca
// Example: networks/net1/ordererOrganizations/net1.com/ca
func (ca *CA) GetSubPath() string {
	if ca.IsOrdererCA() {
		netDomain := fmt.Sprintf("net%d.com", ca.NetworkID)
		return filepath.Join("networks", "net" + strconv.Itoa(ca.NetworkID),
			"ordererOrganizations", netDomain,
			"ca",
		)
	}

	orgDomain := fmt.Sprintf("org%d.net%d.com", ca.OrganizationID, ca.NetworkID)
	return filepath.Join("networks", "net" + strconv.Itoa(ca.NetworkID),
		"peerOrganizations", orgDomain,
		"ca",
	)
}

// Get ca url
// Example: ca.org1.net1.com
// Example: ca.net1.com
func (ca *CA) GetUrl() string {
	if ca.IsOrdererCA() {
		return fmt.Sprintf("ca.net%d.com", ca.NetworkID)
	}

	return fmt.Sprintf("ca.org%d.net%d.com", ca.OrganizationID, ca.NetworkID)
}

func (ca *CA) GetSelector() map[string]string {
	if ca.IsOrdererCA() {
		return map[string]string{
			"app": "mictract",
			"net": strconv.Itoa(ca.NetworkID),
			"org": "orderer",
			"tier": "ca",
		}
	}

	return map[string]string{
		"app": "mictract",
		"net": strconv.Itoa(ca.NetworkID),
		"org": strconv.Itoa(ca.OrganizationID),
		"tier": "ca",
	}
}

func (ca *CA) GetPod() (*corev1.Pod, error) {
	return getPod(ca)
}

// Connect to K8S to create the configMap.
func (ca *CA) CreateConfigMap() {
	name := ca.GetName()

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "-env",
		},
		Data:	map[string]string{
			"FABRIC_CA_HOME": "/etc/hyperledger/fabric-ca-server",
			"FABRIC_CA_SERVER_CA_NAME": name,
			"FABRIC_CA_SERVER_PORT": "7054",
			"FABRIC_CA_SERVER_TLS_ENABLED": "true",
			"FABRIC_CA_SERVER_CSR_HOSTS": name,
		},
	}

	_, err := global.K8sClientset.CoreV1().
		ConfigMaps(corev1.NamespaceDefault).
		Create(context.TODO(), configMap, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create CA config map error", zap.Error(err))
	}
}

// Connect to K8S to create the deployment.
func (ca *CA) CreateDeployment() {
	subPath := ca.GetSubPath()
	name := ca.GetName()

	matchLabel := map[string]string{
		"app": "mictract",
		"net": strconv.Itoa(ca.NetworkID),
		"org": strconv.Itoa(ca.OrganizationID),
		"tier": "ca",
	}

	if ca.IsOrdererCA() {
		matchLabel = map[string]string{
			"app": "mictract",
			"net": strconv.Itoa(ca.NetworkID),
			"org": "orderer",
			"tier": "ca",
		}
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabel,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   name,
					Labels: matchLabel,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "fabric-ca",
							Image: "hyperledger/fabric-ca:1.4.9",
							Command: []string{ "sh", "-c", "fabric-ca-server start -b admin:adminpw -d > /dev/termination-log" },
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: name + "-env",
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "ca",
									Protocol:      corev1.ProtocolTCP,
									ContainerPort: 7054,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:             "data",
									MountPath:        "/etc/hyperledger/fabric-ca-server",
									SubPath:	subPath,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name:         "data",
							VolumeSource: corev1.VolumeSource{
								NFS: &corev1.NFSVolumeSource{
									Server: config.NFS_SERVER_URL,
									Path:   config.NFS_EXPOSED_PATH,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := global.K8sClientset.AppsV1().
		Deployments(corev1.NamespaceDefault).
		Create(context.TODO(), deployment, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create CA deployment error", zap.Error(err))
	}
}

// Connect to K8S to create the service.
func (ca *CA) CreateService() {
	name := ca.GetName()

	selector := map[string]string{
		"app": "mictract",
		"net": strconv.Itoa(ca.NetworkID),
		"org": strconv.Itoa(ca.OrganizationID),
		"tier": "ca",
	}

	if ca.IsOrdererCA() {
		selector = map[string]string{
			"app": "mictract",
			"net": strconv.Itoa(ca.NetworkID),
			"org": "orderer",
			"tier": "ca",
		}
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name: "ca",
					Port: 7054,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "ca",
					},
				},
			},
		},
	}

	_, err := global.K8sClientset.CoreV1().
		Services(corev1.NamespaceDefault).
		Create(context.TODO(), service, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create CA service error", zap.Error(err))
	}
}

// Connect to K8S to create the ingress resource.
func (ca *CA) CreateIngress() {
	name := ca.GetName()
	pathType := netv1.PathTypePrefix

	ingress := &netv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec:       netv1.IngressSpec{
			Rules: []netv1.IngressRule{
				{
					Host: ca.GetUrl(),
					IngressRuleValue: netv1.IngressRuleValue{
						HTTP: &netv1.HTTPIngressRuleValue{
							Paths: []netv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend:  netv1.IngressBackend{
										Service:  &netv1.IngressServiceBackend{
											Name: name,
											Port: netv1.ServiceBackendPort{
												Number: 7054,
											},

										},
									},

								},
							},
						},
					},
				},
			},
		},
	}

	_, err := global.K8sClientset.NetworkingV1().
		Ingresses(corev1.NamespaceDefault).
		Create(context.TODO(), ingress, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create CA ingress error", zap.Error(err))
	}
}

// Connect to K8S to create all the resources.
func (ca *CA) Create() {
	ca.CreateConfigMap()
	ca.CreateDeployment()
	ca.CreateService()
	// ca.CreateIngress(global.K8sClientset)
}

// Connect to K8S to delete all the resources.
func (ca *CA) Delete() {
	var err error
	name := ca.GetName()

	err = global.K8sClientset.CoreV1().
		ConfigMaps(corev1.NamespaceDefault).
		Delete(context.TODO(), name + "-env", metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete CA config map error", zap.Error(err))
	}

	err = global.K8sClientset.AppsV1().
		Deployments(corev1.NamespaceDefault).
		Delete(context.TODO(), name, metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete CA deployment error", zap.Error(err))
	}

	err = global.K8sClientset.CoreV1().
		Services(corev1.NamespaceDefault).
		Delete(context.TODO(), name, metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete CA service error", zap.Error(err))
	}
}

func (ca *CA) Watch() {
	watch(ca, &ca.callback)
}

func (ca *CA) ExecCommand(cmd ...string) (string, string, error) {
	return execCommand(ca, cmd...)
}
