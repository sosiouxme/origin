package origin

import (
	configapi "github.com/openshift/origin/pkg/cmd/server/api"
)

// AssetConfig defines the required parameters for starting the OpenShift master
type AssetConfig struct {
	Options               configapi.AssetConfig
	LimitRequestOverrides *configapi.ClusterResourceOverrideConfig
}

// NewAssetConfig returns a new AssetConfig
func NewAssetConfig(options configapi.AssetConfig, limitRequestOverrides *configapi.ClusterResourceOverrideConfig) (*AssetConfig, error) {
	return &AssetConfig{options, limitRequestOverrides}, nil
}
