package diagnostics

import (
	"fmt"
	"time"

	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	"github.com/openshift/origin/pkg/cmd/cli/config"

	_ "github.com/openshift/origin/pkg/cmd/experimental/diagnostics/options"
	clientdiagnostics "github.com/openshift/origin/pkg/diagnostics/client"
	"github.com/openshift/origin/pkg/diagnostics/log"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

func (o DiagnosticsOptions) detectClientConfig() (bool, []types.DiagnosticError, []types.DiagnosticError) {
	diagnostic := &clientdiagnostics.ConfigLoading{ConfFlagName: config.OpenShiftConfigFlagName, ClientFlags: o.ClientFlags}
	o.Logger.Noticet("diagRun", "Determining if client configuration exists for client/cluster diagnostics",
		log.Hash{"area": "client", "name": diagnostic.Name(), "diag": diagnostic.Description()})
	result := diagnostic.Check()
	for _, entry := range result.Logs() {
		o.Logger.LogEntry(entry)
	}
	return diagnostic.SuccessfulLoad(), result.Warnings(), result.Errors()
}

func (o DiagnosticsOptions) buildRawConfig() (*clientcmdapi.Config, error) {
	kubeConfig, configErr := o.Factory.OpenShiftClientConfig.RawConfig()
	if len(kubeConfig.Contexts) == 0 {
		return nil, configErr
	}
	return &kubeConfig, configErr
}

// TODO: probably don't need this at all

func (o DiagnosticsOptions) testContextConnection() (*clientcmdapi.Config, error) {
	confChan := make(chan *clientcmdapi.Config, 1)
	errChan := make(chan error, 1)
	go func() {
		kubeConfig, configErr := o.Factory.OpenShiftClientConfig.RawConfig()
		if len(kubeConfig.Contexts) > 0 {
			confChan <- &kubeConfig
		} else {
			errChan <- configErr
		}
	}()
	var kubeConfig *clientcmdapi.Config = nil
	var configErr error
	select {
	case kubeConfig = <-confChan:
	case configErr = <-errChan:
	case <-time.After(time.Second * 2):
		configErr = fmt.Errorf("Timed out while trying to load client configuration. This likely means either the connection to check the server timed ")
	}
	return kubeConfig, configErr
}
