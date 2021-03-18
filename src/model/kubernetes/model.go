package kubernetes

import (
	"bytes"
	"fmt"
	"go.uber.org/zap"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/remotecommand"
	"mictract/global"
	"sync"
)

type K8sModel interface {
	GetName()				string
	GetSelector()			map[string]string
	GetPod()				(*apiv1.Pod, error)
	Create()
	AwaitableCreate()		error
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

// TODO: maybe memory leak
// watch function will watching your kubernetes model according to the model labels.
// Kubernetes informer will scan all the resources, when your model status had been changed, it will call the `EventHandler` here.
// watch function here just register your callback as handlers into informer.
//
// Note: this function may cause memory leak, because the `EventHandler` will never be removed.
// 	If your callback closure some large objects, its may wont be collect.
func watch(m K8sModel, cb *callback) {
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

// awaitableCreate provide ability to wait creation process.
// When informer listened to your k8s model and get the change of your model phase, awaitableCreate will return.
// If your model has been running or duplicated, it return nil.
// If your model failed or some unknown causes, it return an error.
func awaitableCreate(m K8sModel) (err error) {
	wg := sync.WaitGroup{}
	cb := func(old apiv1.PodPhase, new apiv1.PodPhase) {
		if new == apiv1.PodRunning {
			wg.Done()
		} else if new == apiv1.PodFailed || new == apiv1.PodUnknown {
			err = fmt.Errorf("error occurred when %s creating", m.GetName())
			wg.Done()
		}
	}

	watch(m, &callback{
		onPodPhaseUpdate: []func(apiv1.PodPhase, apiv1.PodPhase) {
			cb,
		},
	})

	wg.Add(1)
	m.Create()
	wg.Wait()

	return err
}


type callback struct {
	onPodPhaseUpdate []func(apiv1.PodPhase, apiv1.PodPhase)
}

func (c *callback) AddPodPhaseUpdateHandler(handler func(apiv1.PodPhase, apiv1.PodPhase)) {
	c.onPodPhaseUpdate = append(c.onPodPhaseUpdate, handler)
}