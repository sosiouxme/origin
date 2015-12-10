package client

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
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
	Level      int
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
	d.runDiagnosticPod(nil, r)
	return r
}

func (d *DiagnosticPod) runDiagnosticPod(service *kapi.Service, r types.DiagnosticResult) {
	loglevel := d.Level
	if loglevel > 2 {
		loglevel = 2 // need to show summary at least
	}
	pod, err := d.KubeClient.Pods(d.Namespace).Create(&kapi.Pod{
		ObjectMeta: kapi.ObjectMeta{GenerateName: "pod-diagnostic-test"},
		Spec: kapi.PodSpec{
			Containers: []kapi.Container{
				kapi.Container{
					Name:    "pod-diagnostics",
					Image:   "docker.io/openshift/origin",
					Command: []string{"pod-diagnostics", "-l", strconv.Itoa(loglevel)},
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

	bytelim := int64(1024000)
	podLogsOpts := &kapi.PodLogOptions{
		TypeMeta:   pod.TypeMeta,
		Container:  "pod-diagnostics",
		Follow:     true,
		LimitBytes: &bytelim,
	}
	req, err := d.Factory.LogsForObject(pod, podLogsOpts)
	if err != nil {
		r.Error("DCli2005", err, fmt.Sprintf("The request for diagnostic pod logs failed unexpectedly. Error: (%T) %[1]v", err))
		return
	}

	// wait for pod to be scheduled and started, then watch logs and wait until it exits
	readCloser, err := req.Stream()
	for times := 1; err != nil; times++ {
		if times <= 15 { // try, try again
			r.Debug("DCli2010", fmt.Sprintf("Couldn't get pod logs: %[1]v", err))
			time.Sleep(time.Duration(times*100) * time.Millisecond)
			readCloser, err = req.Stream()
			continue
		}
		r.Error("DCli2006", err, fmt.Sprintf("Error preparing diagnostic pod logs for streaming: (%T) %[1]v", err))
		return
	}
	defer readCloser.Close()
	lineScanner := bufio.NewScanner(readCloser)
	podLogs, warnings, errors := "", 0, 0
	errorRegex := regexp.MustCompile(`^\[Note\]\s+Errors\s+seen:\s+(\d+)`)
	warnRegex := regexp.MustCompile(`^\[Note\]\s+Warnings\s+seen:\s+(\d+)`)
	for lineScanner.Scan() {
		line := lineScanner.Text()
		podLogs += line + "\n"
		if matches := errorRegex.FindStringSubmatch(line); matches != nil {
			errors, _ = strconv.Atoi(matches[1])
		} else if matches := warnRegex.FindStringSubmatch(line); matches != nil {
			warnings, _ = strconv.Atoi(matches[1])
		}
	}
	if err := lineScanner.Err(); err != nil { // Scan terminated abnormally
		r.Error("DCli2009", err, fmt.Sprintf("Unexpected error reading diagnostic pod logs: (%T) %[1]v\nLogs are:\n%[2]s", err, podLogs))
	} else {
		if errors > 0 {
			r.Error("DCli2012", nil, "See the errors below in the output from the diagnostic pod:\n"+podLogs)
		} else if warnings > 0 {
			r.Warn("DCli2013", nil, "See the warnings below in the output from the diagnostic pod:\n"+podLogs)
		} else {
			r.Info("DCli2008", "Output from the diagnostic pod:\n"+podLogs)
		}
	}
}
