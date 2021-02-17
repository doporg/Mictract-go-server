package kubernetes

import "k8s.io/client-go/kubernetes"

type K8sModel interface {
	Create(clientset *kubernetes.Clientset)
	Delete(clientset *kubernetes.Clientset)
}
