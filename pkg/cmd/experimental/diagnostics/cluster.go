package diagnostics

import (
	"fmt"
	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"

	clustdiags "github.com/openshift/origin/pkg/diagnostics/cluster"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

var (
	AvailableClusterDiagnostics = util.NewStringSet("NodeDefinitions")
)

func (o DiagnosticsOptions) buildClusterDiagnostics(rawConfig *clientcmdapi.Config) ([]types.Diagnostic, bool /* ok */, error) {
	requestedDiagnostics := intersection(util.NewStringSet(o.RequestedDiagnostics...), AvailableClusterDiagnostics).List()
	if len(requestedDiagnostics) == 0 { // no diagnostics to run here
		return nil, true, nil // don't waste time on discovery
	}

	/* TODO: get admin kubeclient
	kubeConfig, err := o.Factory.OpenShiftClientConfig.RawConfig()
	if err != nil { // failed to obtain full client configuration
		return nil, false, err
	}
	*/

	_, kubeClient, configErr := o.Factory.Clients()
	/*
		if configErr != nil { // failed to create a client...
			return nil, false, configErr
		}
	*/

	diagnostics := []types.Diagnostic{}
	for _, diagnosticName := range requestedDiagnostics {
		switch diagnosticName {
		case "NodeDefinitions":
			diagnostics = append(diagnostics, clustdiags.NodeDefinition{kubeClient})

		default:
			return nil, false, fmt.Errorf("unknown diagnostic: %v", diagnosticName)
		}
	}
	return diagnostics, true, configErr
}
