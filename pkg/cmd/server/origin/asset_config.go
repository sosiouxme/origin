package origin

import (
	configapi "github.com/openshift/origin/pkg/cmd/server/api"
)

// AssetConfig defines the required parameters for starting the OpenShift master
type AssetConfig struct {
	Options               configapi.AssetConfig
	AdmissionPluginConfig map[string]configapi.AdmissionPluginConfig
}

// NewAssetConfig returns a new AssetConfig
func NewAssetConfig(options configapi.AssetConfig, admissionPluginConfig map[string]configapi.AdmissionPluginConfig) (*AssetConfig, error) {
	return &AssetConfig{options, admissionPluginConfig}, nil
}
