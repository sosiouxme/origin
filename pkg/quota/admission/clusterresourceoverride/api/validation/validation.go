package validation

import (
	"fmt"

	"github.com/openshift/origin/pkg/quota/admission/clusterresourceoverride/api"
)

func Validate(config *api.ClusterResourceOverrideConfig) error {
	if config != nil {
		if config.LimitCPUToMemoryPercent == 0.0 && config.CPURequestToLimitPercent == 0.0 && config.MemoryRequestToLimitPercent == 0.0 {
			return fmt.Errorf("%s plugin enabled but no ratios specified", api.PluginName)
		}
		if config.LimitCPUToMemoryPercent < 0.0 {
			return fmt.Errorf("%s.LimitCPUToMemoryPercent must be positive", api.PluginName)
		}
		if config.CPURequestToLimitPercent < 0.0 || config.CPURequestToLimitPercent > 100.0 {
			return fmt.Errorf("%s.CPURequestToLimitPercent must be between 0.0 and 100.0", api.PluginName)
		}
		if config.MemoryRequestToLimitPercent < 0.0 || config.MemoryRequestToLimitPercent > 100.0 {
			return fmt.Errorf("%s.MemoryRequestToLimitPercent must be between 0.0 and 100.0", api.PluginName)
		}
	}
	return nil
}
