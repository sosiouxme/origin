package admission

import (
	"k8s.io/kubernetes/pkg/admission"

	"github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/cmd/server/api"
	"github.com/openshift/origin/pkg/cmd/util/pluginconfig"
	"github.com/openshift/origin/pkg/project/cache"
)

type PluginInitializer struct {
	OpenshiftClient client.Interface
	ProjectCache    *cache.ProjectCache
}

// Initialize will check the initialization interfaces implemented by each plugin
// and provide the appropriate initialization data
func (i *PluginInitializer) Initialize(plugins []admission.Interface) {
	for _, plugin := range plugins {
		if wantsOpenshiftClient, ok := plugin.(WantsOpenshiftClient); ok {
			wantsOpenshiftClient.SetOpenshiftClient(i.OpenshiftClient)
		}
		if wantsProjectCache, ok := plugin.(WantsProjectCache); ok {
			wantsProjectCache.SetProjectCache(i.ProjectCache)
		}
	}
}

// Validate will call the Validate function in each plugin if they implement
// the Validator interface.
func Validate(plugins []admission.Interface) error {
	for _, plugin := range plugins {
		if validater, ok := plugin.(Validator); ok {
			err := validater.Validate()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetPluginConfigFile translates from the master plugin config to a file name containing
// a particular plugin's config (the file may be a temp file if config is embedded)
func GetPluginConfigFile(pluginConfig map[string]api.AdmissionPluginConfig,
	pluginName string, defConfigFile string) (string, error) {
	// Check whether a config is specified for this plugin. If not, default to the
	// global plugin config file (if any).
	if cfg, hasConfig := pluginConfig[pluginName]; hasConfig {
		configFile, err := pluginconfig.GetPluginConfig(cfg)
		if err != nil {
			return "", err
		}
		return configFile, nil
	}
	return defConfigFile, nil
}
