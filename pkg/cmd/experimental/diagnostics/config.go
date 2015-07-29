package diagnostics

import (
	clientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

func (o DiagnosticsOptions) buildRawConfig() (*clientcmdapi.Config, error) {
}

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
	return &kubeConfig, configErr
}
