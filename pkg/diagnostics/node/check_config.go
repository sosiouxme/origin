package node

import (
	"errors"

	configapilatest "github.com/openshift/origin/pkg/cmd/server/api/latest"
	configvalidation "github.com/openshift/origin/pkg/cmd/server/api/validation"
	"github.com/openshift/origin/pkg/diagnostics/log"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

// NodeConfigCheck
type NodeConfigCheck struct {
	NodeConfigFile string

	Log *log.Logger
}

func (d NodeConfigCheck) Description() string {
	return "Check the node config file"
}
func (d NodeConfigCheck) CanRun() (bool, error) {
	if len(d.NodeConfigFile) == 0 {
		return false, errors.New("must have node config file")
	}

	return true, nil
}
func (d NodeConfigCheck) Check() *types.DiagnosticResult {
	r := &types.DiagnosticResult{}
	d.Log.Debugf("discNCfile", "Looking for node config file at '%s'", d.NodeConfigFile)
	nodeConfig, err := configapilatest.ReadAndResolveNodeConfig(d.NodeConfigFile)
	if err != nil {
		d.Log.Errorf("discNCfail", "Could not read node config file '%s':\n(%T) %[2]v", d.NodeConfigFile, err)
		return r.Error(err)
	}

	d.Log.Infof("discNCfound", "Found a node config file:\n%[1]s", d.NodeConfigFile)

	return r.Error(configvalidation.ValidateNodeConfig(nodeConfig)...)
}
