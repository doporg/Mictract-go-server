package kubernetes

import (
	"context"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"mictract/config"
	"mictract/global"
)

type Mysql struct {
	callback
}

func NewMysql() *Mysql {
	return &Mysql{}
}

// Get peer name.
// Example: peer1-org1-net1
func (m *Mysql) GetName() string {
	return "mysql"
}



func (m *Mysql) GetSelector() map[string]string {
	return map[string]string{
		"app": "mictract",
		"tier": "database",
	}
}

func (m *Mysql) GetPod() (*apiv1.Pod, error) {
	return getPod(m)
}

// Connect to K8S to create the configMap.
func (m *Mysql) CreateConfigMap() {
	name := m.GetName()

	configMap := &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: name + "-env",
		},
		Data:	map[string]string{
			"MYSQL_ROOT_PASSWORD": config.DB_PW,
		},
	}

	_, err := global.K8sClientset.CoreV1().
		ConfigMaps(apiv1.NamespaceDefault).
		Create(context.TODO(), configMap, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create mysql config map error", zap.Error(err))
	}
}

// Connect to K8S to create the deployment.
func (m *Mysql) CreateDeployment() {
	name := m.GetName()

	matchlabels := map[string]string{
		"app": "mictract",
		"tier": "mysql",
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
							Name:  "mysql",
							Image: "mysql:5.7",
							//Command: []string{ "sleep", "infinity" },
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
									Name:          "mysql",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 3306,
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:             "mysql",
									MountPath:        "/var/lib/mysql",
									SubPath: 		"mysql",
								},
							},
							Args: []string{
								"--ignore-db-dir=lost+found",
							},
						},
					},
					Volumes: []apiv1.Volume{
						{
							Name:         "mysql",
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
		global.Logger.Error("Create mysql deployment error", zap.Error(err))
	}
}

// Connect to K8S to create the service.
func (m *Mysql) CreateService() {
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mysql",
		},
		Spec: apiv1.ServiceSpec{
			Type: apiv1.ServiceTypeClusterIP,
			Selector: map[string]string{
				"app": "mictract",
				"tier": "mysql",
			},
			Ports: []apiv1.ServicePort{
				{
					Name: "mysql",
					Port: 3306,
					TargetPort: intstr.IntOrString{
						Type:   intstr.String,
						StrVal: "mysql",
					},
				},
			},
		},
	}

	_, err := global.K8sClientset.CoreV1().
		Services(apiv1.NamespaceDefault).
		Create(context.TODO(), service, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create mysql service error", zap.Error(err))
	}
}

// Connect to K8S to create all the resources.
func (m *Mysql) Create() {
	m.CreateConfigMap()
	m.CreateDeployment()
	m.CreateService()
}

// Connect to K8S to delete all the resources.
func (m *Mysql) Delete() {
	var err error
	name := m.GetName()

	err = global.K8sClientset.CoreV1().
		ConfigMaps(apiv1.NamespaceDefault).
		Delete(context.TODO(), name + "-env", metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete mysql config map error", zap.Error(err))
	}

	err = global.K8sClientset.AppsV1().
		Deployments(apiv1.NamespaceDefault).
		Delete(context.TODO(), name, metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete mysql deployment error", zap.Error(err))
	}

	err = global.K8sClientset.CoreV1().
		Services(apiv1.NamespaceDefault).
		Delete(context.TODO(), name, metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete mysql service error", zap.Error(err))
	}
}

func (m *Mysql) Watch() {
	watch(m, &m.callback)
}

func (m *Mysql) ExecCommand(cmd ...string) (string, string, error) {
	return execCommand(m, cmd...)
}

