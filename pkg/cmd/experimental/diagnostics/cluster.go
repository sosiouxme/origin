package diagnostics

import (
	"fmt"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	clientdiagnostics "github.com/openshift/origin/pkg/diagnostics/client"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

var (
	AvailableClusterDiagnostics = util.NewStringSet("NodeDefinitions")
)

func (o DiagnosticsOptions) buildClusterDiagnostics() ([]types.Diagnostic, bool /* ok */, error) {
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

	_, kubeClient, err := o.Factory.Clients()
	if err != nil { // failed to create a client...
		return nil, false, err
	}

	diagnostics := []types.Diagnostic{}
	for _, diagnosticName := range requestedDiagnostics {
		switch diagnosticName {
		case "NodeDefinitions":
			diagnostics = append(diagnostics, clientdiagnostics.NodeDefinition{kubeClient})

		default:
			return nil, false, fmt.Errorf("unknown diagnostic: %v", diagnosticName)
		}
	}
	return diagnostics, true, nil
}
