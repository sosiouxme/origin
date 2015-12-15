package pod

import (
	"fmt"
	"time"

	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/diagnostics/types"

	kclientcmd "k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
)

const (
	PodCheckAuthName            = "PodCheckAuth"
	StandardSAKubeConfig string = "/var/run/secrets/kubernetes.io/serviceaccount/.kubeconfig"
)

// PodCheckAuth is a Diagnostic to check that a pod can authenticate as expected
type PodCheckAuth struct {
}

func (d PodCheckAuth) Name() string {
	return PodCheckAuthName
}
func (d PodCheckAuth) Description() string {
	return "Check that service account credentials authenticate as expected"
}
func (d PodCheckAuth) CanRun() (bool, error) {
	return true, nil
}

func (d PodCheckAuth) Check() types.DiagnosticResult {
	r := types.NewDiagnosticResult(PodCheckAuthName)
	kubeconfig, err := kclientcmd.LoadFromFile(StandardSAKubeConfig)
	if err != nil {
		r.Error("DP1001", err, fmt.Sprintf("could not load the service account client config: %v", err))
		return r
	}
	clientConfig := kclientcmd.NewDefaultClientConfig(*kubeconfig, &kclientcmd.ConfigOverrides{})
	oclient, _, err := clientcmd.NewFactory(clientConfig).Clients()
	if err != nil {
		r.Error("DP1002", err, fmt.Sprintf("could not create API clients from the service account client config: %v", err))
		return r
	}
	rchan := make(chan error, 1) // for concurrency with timeout
	go func() {
		_, err := oclient.Users().Get("~")
		rchan <- err
	}()

	select {
	case <-time.After(time.Second * 4): // timeout per query
		r.Warn("DP1005", nil, "A request to the master timed out.\nThis could be temporary but could also indicate network or DNS problems.")
	case err := <-rchan:
		if err != nil {
			r.Error("DP1003", err, fmt.Sprintf("Could not authenticate to the master with the service account credentials: %v", err))
		} else {
			r.Debug("DP1004", "Successfully authenticated to master")
		}
	}
	return r
}
