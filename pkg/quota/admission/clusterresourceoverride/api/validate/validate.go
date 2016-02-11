package validate

import (
	"fmt"

	"github.com/openshift/origin/pkg/quota/admission/clusterresourceoverride/api"
)

const pluginname = "ClusterResourceOverride"

func Validate(config *api.ClusterResourceOverrideConfig) error {
	if config != nil {
		if config.LimitCPUToMemoryPercent == 0.0 && config.CPURequestToLimitPercent == 0.0 && config.MemoryRequestToLimitPercent == 0.0 {
			return fmt.Errorf("%s plugin enabled but no ratios specified", pluginname)
		}
		if config.LimitCPUToMemoryPercent < 0.0 {
			return fmt.Errorf("%s.LimitCPUToMemoryPercent must be positive", pluginname)
		}
		if config.CPURequestToLimitPercent < 0.0 || config.CPURequestToLimitPercent > 100.0 {
			return fmt.Errorf("%s.CPURequestToLimitPercent must be between 0.0 and 100.0", pluginname)
		}
		if config.MemoryRequestToLimitPercent < 0.0 || config.MemoryRequestToLimitPercent > 100.0 {
			return fmt.Errorf("%s.MemoryRequestToLimitPercent must be between 0.0 and 100.0", pluginname)
		}
	}
	return nil
}
