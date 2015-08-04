package diagnostics

import (
	"fmt"

	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"

	clientdiagnostics "github.com/openshift/origin/pkg/diagnostics/client"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

var (
	AvailableClientDiagnostics = util.NewStringSet("ConfigContexts")
)

func (o DiagnosticsOptions) buildClientDiagnostics(rawConfig *clientcmdapi.Config) ([]types.Diagnostic, bool /* ok */, error) {

	/* will need this once some client diagnostics actually require a client
	_, kubeClient, err := o.Factory.Clients()
	if err != nil { // failed to create a client...
		return nil, false, err
	}
	*/

	diagnostics := []types.Diagnostic{}
	requestedDiagnostics := intersection(util.NewStringSet(o.RequestedDiagnostics...), AvailableClientDiagnostics).List()
	for _, diagnosticName := range requestedDiagnostics {
		switch diagnosticName {
		case "ConfigContexts":
			for contextName, _ := range rawConfig.Contexts {
				diagnostics = append(diagnostics, clientdiagnostics.ConfigContext{rawConfig, contextName})
			}

		default:
			return nil, false, fmt.Errorf("unknown diagnostic: %v", diagnosticName)
		}
	}
	return diagnostics, true, nil
}
