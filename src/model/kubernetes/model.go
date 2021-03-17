package kubernetes

import (
	"bytes"
	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/remotecommand"
	"mictract/global"
)

type K8sModel interface {
	GetSelector()			map[string]string
	GetPod()				(*apiv1.Pod, error)
	Create()
	Delete()
	Watch()
	ExecCommand(...string)	(string, string, error)
}

func getPod(m K8sModel) (*apiv1.Pod, error) {
	pods, err := global.K8sLister.List(labels.Set(m.GetSelector()).AsSelector())

	if err != nil {
		return nil, err
	}

	if len(pods) == 0 {
		return nil, nil
	}

	return pods[0], nil
}

func watch(m K8sModel, cb *callback) {
	// func watch(m K8sModel, phaseUpdate []func(apiv1.PodPhase, apiv1.PodPhase)) {
	labelContains := func (target map[string]string) bool {
		for key, val := range m.GetSelector() {
			if targetVal, ok := target[key]; !ok || val != targetVal {
				return false
			}
		}
		return true
	}

	global.K8sInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(new interface{}) {
			pod := new.(*apiv1.Pod)
			if !labelContains(pod.Labels) { return }

			for _, fn := range cb.onPodPhaseUpdate {
				phase := pod.Status.Phase
				fn(phase, phase)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			oldPod, newPod := old.(*apiv1.Pod), new.(*apiv1.Pod)
			if !labelContains(newPod.Labels) { return }

			for _, fn := range cb.onPodPhaseUpdate {
				oldPhase, newPhase := oldPod.Status.Phase, newPod.Status.Phase
				if oldPhase != newPhase {
					fn(oldPhase, newPhase)
				}
			}
		},
	})
}

func execCommand(m K8sModel, cmd ...string) (string, string, error) {
	var podName string
	if pod, err := m.GetPod(); err != nil {
		return "", "", err
	} else {
		podName = pod.Name
	}

	req := global.K8sClientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(apiv1.NamespaceDefault).
		SubResource("exec").
		VersionedParams(&apiv1.PodExecOptions{
			Command:   cmd,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(global.K8sRestConfig, "POST", req.URL())
	if err != nil {
		return "", "", err
	}

	var stdout, stderr bytes.Buffer
	if err = exec.Stream(remotecommand.StreamOptions{
		Stdin:             nil,
		Stdout:            &stdout,
		Stderr:            &stderr,
	}); err != nil {
		global.Logger.Error("Error occurred when exec command",
			zap.String("stdout", stdout.String()),
			zap.String("stderr", stderr.String()),
		)

		return stdout.String(), stderr.String(), err
	}

	return stdout.String(), stderr.String(), nil
}


type callback struct {
	onPodPhaseUpdate []func(apiv1.PodPhase, apiv1.PodPhase)
}

func (c *callback) AddPodPhaseUpdateHandler(handler func(apiv1.PodPhase, apiv1.PodPhase)) {
	c.onPodPhaseUpdate = append(c.onPodPhaseUpdate, handler)
}