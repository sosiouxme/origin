package clusterresourceoverride

import (
	"bytes"
	"fmt"
	"io"
	"reflect"

	oadmission "github.com/openshift/origin/pkg/cmd/server/admission"
	"github.com/openshift/origin/pkg/cmd/server/api"
	"github.com/openshift/origin/pkg/project/cache"
	"k8s.io/kubernetes/pkg/admission"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/plugin/pkg/admission/limitranger"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"speter.net/go/exp/math/dec/inf"
)

const (
	clusterResourceOverrideAnnotation = "quota.openshift.io/cluster-resource-override-enabled"
	cpuBaseScaleFactor                = 1000.0 / (1024.0 * 1024.0 * 1024.0) // 1000 milliCores per 1GiB
)

func init() {
	admission.RegisterPlugin("ClusterResourceOverride", func(client kclient.Interface, config io.Reader) (admission.Interface, error) {
		return newClusterResourceOverride(client, config)
	})
}

type clusterResourceOverridePlugin struct {
	*admission.Handler
	Config       api.ClusterResourceOverrideConfig
	ProjectCache *cache.ProjectCache
	LimitRanger  admission.Interface
}

var _ = oadmission.WantsProjectCache(&clusterResourceOverridePlugin{})
var _ = oadmission.Validator(&clusterResourceOverridePlugin{})

// newClusterResourceOverride returns an admission controller for containers that
// configurably overrides container resource request/limits
func newClusterResourceOverride(client kclient.Interface, config io.Reader) (admission.Interface, error) {
	glog.V(5).Infof("ClusterResourceOverride admission controller is loaded")
	parsed, err := ReadConfig(config)
	glog.V(5).Infof("ClusterResourceOverride admission controller got config: %v\nerror: (%T) %[2]v", parsed, err)
	return &clusterResourceOverridePlugin{
		Handler:     admission.NewHandler(admission.Create),
		Config:      parsed,
		LimitRanger: limitranger.NewLimitRanger(client, wrapLimit),
	}, err
}

func wrapLimit(limitRange *kapi.LimitRange, resourceName string, obj runtime.Object) error {
	limitranger.Limit(limitRange, resourceName, obj)
	// always return success so that all defaults will be applied.
	// validation will occur after the overrides.
	return nil
}

func (a *clusterResourceOverridePlugin) SetProjectCache(projectCache *cache.ProjectCache) {
	a.ProjectCache = projectCache
}

func ReadConfig(configFile io.Reader) (api.ClusterResourceOverrideConfig, error) {
	config := api.ClusterResourceOverrideConfig{}
	if configFile == nil || reflect.ValueOf(configFile).IsNil() /* pointer to nil */ {
		glog.V(5).Infof("ClusterResourceOverride has no config to read.")
		return config, nil
	}
	glog.V(5).Infof("ClusterResourceOverride about to read config:\n%v", configFile)
	buffer := new(bytes.Buffer)
	if _, err := buffer.ReadFrom(configFile); err != nil {
		return config, err
	}
	err := yaml.Unmarshal(buffer.Bytes(), &config)
	glog.V(5).Infof("ClusterResourceOverride config:\n%v\nerror: %v", config, err)
	return config, err
}

func Validate(config api.ClusterResourceOverrideConfig) error {
	if config.Enabled {
		if config.LimitCPUToMemoryRatio == 0.0 && config.CPURequestToLimitRatio == 0.0 && config.MemoryRequestToLimitRatio == 0.0 {
			return fmt.Errorf("ClusterResourceOverride plugin enabled but no ratios specified")
		}
		if config.LimitCPUToMemoryRatio < 0.0 {
			return fmt.Errorf("LimitCPUToMemoryRatio must be positive")
		}
		if config.CPURequestToLimitRatio < 0.0 || config.CPURequestToLimitRatio > 1.0 {
			return fmt.Errorf("CPURequestToLimitRatio must be between 0.0 and 1.0")
		}
		if config.MemoryRequestToLimitRatio < 0.0 || config.MemoryRequestToLimitRatio > 1.0 {
			return fmt.Errorf("MemoryRequestToLimitRatio must be between 0.0 and 1.0")
		}
	}
	return nil
}
func (a *clusterResourceOverridePlugin) Validate() error {
	if err := Validate(a.Config); err != nil {
		return err
	}
	if a.ProjectCache == nil {
		return fmt.Errorf("ClusterResourceOverride did not get a project cache")
	}
	return nil
}

// TODO this will need to update when we have pod requests/limits
func (a *clusterResourceOverridePlugin) Admit(attr admission.Attributes) error {
	glog.V(8).Infof("ClusterResourceOverride admission controller is invoked")
	if !a.Config.Enabled || attr.GetResource() != kapi.Resource("pods") || attr.GetSubresource() != "" {
		return nil // not applicable
	}
	pod, ok := attr.GetObject().(*kapi.Pod)
	if !ok {
		return admission.NewForbidden(attr, fmt.Errorf("unexpected object: %#v", attr.GetObject()))
	}
	glog.V(5).Infof("ClusterResourceOverride is looking at creating pod %s in project %s", pod.Name, attr.GetNamespace())

	// allow annotations on project to override
	if ns, err := a.ProjectCache.GetNamespace(attr.GetNamespace()); err != nil {
		glog.Warningf("ClusterResourceOverride got an error retrieving namespace: %v", err)
		return admission.NewForbidden(attr, err) // this should not happen though
	} else {
		projectEnabledPlugin, exists := ns.Annotations[clusterResourceOverrideAnnotation]
		if exists && projectEnabledPlugin != "true" {
			glog.V(5).Infof("ClusterResourceOverride is disabled for project %s", attr.GetNamespace())
			return nil // disabled for this project, do nothing
		}
	}

	// Reuse LimitRanger logic to apply limit/req defaults from the project. Ignore validation
	// errors, assume that LimitRanger will run after this plugin to validate.
	glog.V(5).Infof("ClusterResourceOverride: initial pod limits are: %#v", pod.Spec.Containers[0].Resources)
	if err := a.LimitRanger.Admit(attr); err != nil {
		glog.V(5).Infof("ClusterResourceOverride: error from LimitRanger: %#v", err)
	}
	pod, ok = attr.GetObject().(*kapi.Pod)
	if !ok {
		return admission.NewForbidden(attr, fmt.Errorf("unexpected object: %#v", attr.GetObject()))
	}
	glog.V(5).Infof("ClusterResourceOverride: pod limits after LimitRanger are: %#v", pod.Spec.Containers[0].Resources)
	for _, container := range pod.Spec.Containers {
		resources := container.Resources
		memLimit := resources.Limits.Memory()
		// Although resource Quantity objects are used for both memory and CPU they're at a very
		// different scale. The inf.Dec number type is for decimal numbers down to 0.001,
		// stored as ints with a scale. Memory is measured in bytes, while CPU is in cores,
		// with 3 places of accuracy for millicores. In order to maintain 3 places of accuracy
		// for CPU there is some hocus-pocus with the "scale" component when multiplying
		// (inf.Dec does not have a multiplication method).
		if a.Config.LimitCPUToMemoryRatio != 0.0 {
			resources.Limits[kapi.ResourceCPU] = resource.Quantity{
				Amount: inf.NewDec(int64(float64(memLimit.Value())*a.Config.LimitCPUToMemoryRatio*cpuBaseScaleFactor), 3),
				Format: resources.Limits.Cpu().Format,
			}
		}
		if a.Config.CPURequestToLimitRatio != 0.0 {
			resources.Requests[kapi.ResourceCPU] = resource.Quantity{
				Amount: inf.NewDec(int64(float64(resources.Limits.Cpu().MilliValue())*a.Config.CPURequestToLimitRatio), 3),
				Format: resources.Requests.Cpu().Format,
			}
		}
		if a.Config.MemoryRequestToLimitRatio != 0.0 {
			resources.Requests[kapi.ResourceMemory] = resource.Quantity{
				Amount: inf.NewDec(int64(float64(resources.Limits.Memory().Value())*a.Config.MemoryRequestToLimitRatio), 0),
				Format: resources.Requests.Memory().Format,
			}
		}
	}
	glog.V(5).Infof("ClusterResourceOverride: pod limits after overrides are: %#v", pod.Spec.Containers[0].Resources)
	return nil
}
