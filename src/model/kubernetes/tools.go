package kubernetes

import (
	"context"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"mictract/config"
	"mictract/global"
)

type Tools struct {}

func (t *Tools) GetName() string {
	return "tools"
}

// Connect to K8S to create the deployment.
func (t *Tools) CreateDeployment(clientset *kubernetes.Clientset) {
	name := t.GetName()

	matchLabels := map[string]string{
		"app": "mictract",
		"tier": "tools",
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: matchLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					Labels: matchLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tools",
							Image: "hyperledger/fabric-tools:2.2.1",
							Command: []string{ "sleep", "infinity" },
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:             "networks",
									MountPath:        "/mictract",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name:         "networks",
							VolumeSource: corev1.VolumeSource{
								NFS: &corev1.NFSVolumeSource{
									Server:   "192.168.1.4",
									Path: config.NFS_EXPOSED_PATH,
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := clientset.AppsV1().
		Deployments(corev1.NamespaceDefault).
		Create(context.TODO(), deployment, metav1.CreateOptions{})

	if err != nil {
		global.Logger.Error("Create tools deployment error", zap.Error(err))
	}
}

// Connect to K8S to create all the resources.
func (t *Tools) Create(clientset *kubernetes.Clientset) {
	t.CreateDeployment(clientset)
}

// Connect to K8S to delete all the resources.
func (t *Tools) Delete(clientset *kubernetes.Clientset) {
	var err error
	name := t.GetName()

	err = clientset.AppsV1().
		Deployments(corev1.NamespaceDefault).
		Delete(context.TODO(), name, metav1.DeleteOptions{})

	if err != nil {
		global.Logger.Error("Delete tools deployment error", zap.Error(err))
	}
}
