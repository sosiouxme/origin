package discovery

import (
	"github.com/openshift/origin/pkg/cmd/server/start"
	osclientcmd "github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/diagnostics/log"
	"github.com/openshift/origin/pkg/diagnostics/types"
	"os/exec"
	"runtime"
)

// ----------------------------------------------------------
// Examine system and return findings in an Environment
func Run(fl *types.Flags, f *osclientcmd.Factory) (env *types.Environment, ok bool) {
	log.Notice("discBegin", "Beginning discovery of environment")
	env = types.NewEnvironment(fl)
	operatingSystemDiscovery(env)
	if fl.CanCheck[types.MasterTarget] || fl.CanCheck[types.NodeTarget] {
		discoverSystemd(env)
	}
	if fl.CanCheck[types.MasterTarget] {
		if fl.MasterOptions == nil {
			fl.MasterOptions = &start.MasterOptions{ConfigFile: fl.MasterConfigPath}
			// no MasterArgs signals it has to be a master config file or nothing.
		} else if fl.MasterConfigPath != "" {
			fl.MasterOptions.ConfigFile = fl.MasterConfigPath
		}
		masterDiscovery(env, fl.MasterOptions)
	}
	if fl.CanCheck[types.NodeTarget] {
		if fl.NodeOptions == nil {
			fl.NodeOptions = &start.NodeOptions{ConfigFile: fl.NodeConfigPath}
			// no NodeArgs signals it has to be a node config file or nothing.
		} else if fl.NodeConfigPath != "" {
			fl.NodeOptions.ConfigFile = fl.NodeConfigPath
		}
		nodeDiscovery(env, fl.NodeOptions)
	}
	if fl.CanCheck[types.ClientTarget] {
		clientDiscovery(env, f)
		readClientConfigFiles(env) // so user knows where config is coming from (or not)
		configClient(env)
	}
	checkAny := false
	for _, check := range fl.CanCheck {
		checkAny = checkAny || check
	}
	if !checkAny {
		if fl.MustCheck == "" {
			log.Error("discNoChecks", "Cannot find any OpenShift configuration. Please specify which component or configuration you wish to troubleshoot.")
		} else {
			log.Errorf("discNoChecks", "Could not find your OpenShift %s configuration for troubleshooting.", fl.MustCheck)
		}
		return env, false
	}
	return env, true
}

// ----------------------------------------------------------
// Determine what we need to about the OS
func operatingSystemDiscovery(env *types.Environment) {
	env.OS = runtime.GOOS
	if env.OS == "linux" {
		if _, err := exec.LookPath("systemctl"); err == nil {
			env.HasSystemd = true
		}
		if _, err := exec.LookPath("/bin/bash"); err == nil {
			env.HasBash = true
		}
	}
}
