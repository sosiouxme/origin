package pod

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/openshift/origin/pkg/cmd/flagtypes"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/diagnostics/types"

	kclient "k8s.io/kubernetes/pkg/client/unversioned"
)

const (
	PodCheckAuthName = "PodCheckAuth"
)

// PodCheckAuth is a Diagnostic to check that a pod can authenticate as expected
type PodCheckAuth struct {
	MasterUrl    string
	MasterCaPath string
	TokenPath    string
}

// Name is part of the Diagnostic interface and just returns name.
func (d PodCheckAuth) Name() string {
	return PodCheckAuthName
}

// Description is part of the Diagnostic interface and provides a user-focused description of what the diagnostic does.
func (d PodCheckAuth) Description() string {
	return "Check that service account credentials authenticate as expected"
}

// CanRun is part of the Diagnostic interface; it determines if the conditions are right to run this diagnostic.
func (d PodCheckAuth) CanRun() (bool, error) {
	return true, nil
}

// Check is part of the Diagnostic interface; it runs the actual diagnostic logic
func (d PodCheckAuth) Check() types.DiagnosticResult {
	r := types.NewDiagnosticResult(PodCheckAuthName)
	token, err := ioutil.ReadFile(d.TokenPath)
	if err != nil {
		r.Error("DP1001", err, fmt.Sprintf("could not read the service account token: %v", err))
		return r
	}
	clientConfig := &clientcmd.Config{
		MasterAddr:     flagtypes.Addr{Value: d.MasterUrl}.Default(),
		KubernetesAddr: flagtypes.Addr{Value: d.MasterUrl}.Default(),
		CommonConfig: kclient.Config{
			TLSClientConfig: kclient.TLSClientConfig{CAFile: d.MasterCaPath},
			BearerToken:     string(token),
		},
	}
	oclient, _, err := clientConfig.Clients()
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
