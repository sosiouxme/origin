package diagnostics

import (
	"fmt"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"

	clientdiagnostics "github.com/openshift/origin/pkg/diagnostics/client"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

var (
	AvailableClientDiagnostics = util.NewStringSet("ConfigContexts")
)

func (o DiagnosticsOptions) buildClientDiagnostics() ([]types.Diagnostic, bool /* ok */, error) {

	/* will need this once some client diagnostics actually require a client
	_, kubeClient, err := o.Factory.Clients()
	if err != nil { // failed to create a client...
		return nil, false, err
	}
	*/

	kubeConfig, configErr := o.Factory.OpenShiftClientConfig.RawConfig()
	/*
		if kubeConfig == nil { // failed to obtain any client configuration
			return nil, false, err
		} // if there was an error, still run the diagnostic to interpret it.
	*/

	diagnostics := []types.Diagnostic{}
	requestedDiagnostics := intersection(util.NewStringSet(o.RequestedDiagnostics...), AvailableClientDiagnostics).List()
	for _, diagnosticName := range requestedDiagnostics {
		switch diagnosticName {
		case "ConfigContexts":
			for contextName, _ := range kubeConfig.Contexts {
				diagnostics = append(diagnostics, clientdiagnostics.ConfigContext{&kubeConfig, contextName})
			}

		default:
			return nil, false, fmt.Errorf("unknown diagnostic: %v", diagnosticName)
		}
	}
	return diagnostics, true, configErr
}
