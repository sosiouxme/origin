package discovery

import (
	osclientcmd "github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/diagnostics/log"
	"github.com/openshift/origin/pkg/diagnostics/types"
	"os/exec"
	"runtime"
)

// ----------------------------------------------------------
// Examine system and return findings in an Environment
func Run(fl *types.Flags, f *osclientcmd.Factory) *types.Environment {
	log.Notice("discBegin", "Beginning discovery of environment")
	env := types.NewEnvironment(fl)
	if config, err := f.OpenShiftClientConfig.RawConfig(); err != nil {
		log.Errorf("discCCstart", "Could not read client config: (%T) %[1]v", err)
	} else {
		env.OsConfig = &config
		env.FactoryForContext[config.CurrentContext] = f
	}
	// run discovery
	operatingSystemDiscovery(env)
	clientDiscovery(env)
	discoverSystemd(env)
	readClientConfigFiles(env) // so user knows where config is coming from (or not)
	configClient(env)
	return env
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
