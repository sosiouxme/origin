package pod

import (
	"fmt"

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
	// TODO: set a timeout
	name, err := oclient.Users().Get("~")
	if err != nil {
		r.Error("DP1003", err, fmt.Sprintf("Could not authenticate to the master with the service account credentials: %v", err))
	} else {
		r.Debug("DP1004", fmt.Sprintf("Successfully authenticated to master as %s", name))
	}

	return r
}
