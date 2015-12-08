package client

import (
	"fmt"
	"time"

	kapi "k8s.io/kubernetes/pkg/api"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"

	osclientcmd "github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

const (
	DiagnosticPodName = "DiagnosticPod"
)

// ConfigLoading is a little special in that it is run separately as a precondition
// in order to determine whether we can run other dependent diagnostics.
type DiagnosticPod struct {
	KubeClient kclient.Client
	Namespace  string
	Factory    *osclientcmd.Factory
}

func (d *DiagnosticPod) Name() string {
	return DiagnosticPodName
}

func (d *DiagnosticPod) Description() string {
	return "Create a pod to run diagnostics from the application standpoint"
}

func (d *DiagnosticPod) CanRun() (bool, error) {
	return true, nil
}

func (d *DiagnosticPod) Check() types.DiagnosticResult {
	r := types.NewDiagnosticResult("DiagnosticPod")
	return r
}

func (d *DiagnosticPod) runDiagnosticPod(service *kapi.Service, r types.DiagnosticResult) {
	pod, err := d.KubeClient.Pods(d.Namespace).Create(&kapi.Pod{
		ObjectMeta: kapi.ObjectMeta{GenerateName: "pod-diagnostic-test"},
		Spec: kapi.PodSpec{
			Containers: []kapi.Container{
				kapi.Container{
					Name:    "pod-diagnostics",
					Image:   "docker.io/openshift/origin",
					Command: []string{"pod-diagnostics"},
				},
			},
		},
	})
	if err != nil {
		r.Error("DCli2001", err, fmt.Sprintf("Creating diagnostic pod failed. Error: (%T) %[1]v", err))
		return
	}
	defer func() { // delete what we created, or notify that we couldn't
		zero := int64(0)
		delOpts := kapi.DeleteOptions{pod.TypeMeta, &zero}
		if err := d.KubeClient.Pods(d.Namespace).Delete(pod.ObjectMeta.Name, &delOpts); err != nil {
			r.Error("DCl2002", err, fmt.Sprintf("Deleting diagnostic pod '%s' failed. Error: %s", pod.ObjectMeta.Name, fmt.Sprintf("(%T) %[1]s", err)))
		}
	}()
	pod, err = d.KubeClient.Pods(d.Namespace).Get(pod.ObjectMeta.Name) // status is filled in post-create
	if err != nil {
		r.Error("DCli2003", err, fmt.Sprintf("Getting created test Pod failed. Error: (%T) %[1]v", err))
		return
	}
	r.Debug("DCli2004", fmt.Sprintf("Created test Pod: %[1]v", pod.ObjectMeta.Name))

	// wait for pod to be scheduled and started, then watch logs and wait until it exits
	podLogs, bytelim := "", int64(1024000)
	podLogsOpts := &kapi.PodLogOptions{
		TypeMeta:   pod.TypeMeta,
		Container:  "origin",
		Follow:     true,
		LimitBytes: &bytelim,
	}
	for times := 1; ; times++ {
		req, err := d.Factory.LogsForObject(pod, podLogsOpts)
		if err != nil {
			if times <= 10 { // try, try again
				//time.Sleep(times * time.Millisecond)
				time.Sleep(time.Duration(times*100) * time.Millisecond)
				continue
			}
			r.Error("DCli2005", err, fmt.Sprintf("Test Pod did not get to a state where logs could be retrieved. Error: (%T) %[1]v", err))
			return
		}
		// TODO got the request object back; use it
		_, _ = podLogs, req
		break
	}

}
