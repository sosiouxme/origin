package clusterresourceoverride

import (
	"fmt"
	"io"
	"io/ioutil"
	"reflect"

	oadmission "github.com/openshift/origin/pkg/cmd/server/admission"
	configlatest "github.com/openshift/origin/pkg/cmd/server/api/latest"
	"github.com/openshift/origin/pkg/project/cache"
	"github.com/openshift/origin/pkg/quota/admission/clusterresourceoverride/api"
	"github.com/openshift/origin/pkg/quota/admission/clusterresourceoverride/api/validation"
	"k8s.io/kubernetes/pkg/admission"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/plugin/pkg/admission/limitranger"

	"github.com/golang/glog"
	"speter.net/go/exp/math/dec/inf"
)

const (
	clusterResourceOverrideAnnotation = "quota.openshift.io/cluster-resource-override-enabled"
	cpuBaseScaleFactor                = 1000.0 / (1024.0 * 1024.0 * 1024.0) // 1000 milliCores per 1GiB
)

func init() {
	admission.RegisterPlugin(api.PluginName, func(client kclient.Interface, config io.Reader) (admission.Interface, error) {
		return newClusterResourceOverride(client, config)
	})
}

type clusterResourceOverridePlugin struct {
	*admission.Handler
	Config       *api.ClusterResourceOverrideConfig
	ProjectCache *cache.ProjectCache
	LimitRanger  admission.Interface
}

var _ = oadmission.WantsProjectCache(&clusterResourceOverridePlugin{})
var _ = oadmission.Validator(&clusterResourceOverridePlugin{})

// newClusterResourceOverride returns an admission controller for containers that
// configurably overrides container resource request/limits
func newClusterResourceOverride(client kclient.Interface, config io.Reader) (admission.Interface, error) {
	parsed, err := ReadConfig(config)
	if err != nil {
		glog.V(5).Infof("%s admission controller loaded with error: (%T) %[2]v", api.PluginName, err)
		return nil, err
	}
	glog.V(5).Infof("%s admission controller loaded with config: %v", api.PluginName, parsed)
	return &clusterResourceOverridePlugin{
		Handler:     admission.NewHandler(admission.Create),
		Config:      parsed,
		LimitRanger: limitranger.NewLimitRanger(client, wrapLimit),
	}, nil
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

func ReadConfig(configFile io.Reader) (*api.ClusterResourceOverrideConfig, error) {
	if configFile == nil || reflect.ValueOf(configFile).IsNil() /* pointer to nil */ {
		glog.V(5).Infof("%s has no config to read.", api.PluginName)
		return nil, nil
	}
	configBytes, err := ioutil.ReadAll(configFile)
	if err != nil {
		return nil, err
	}

	config := &api.ClusterResourceOverrideConfig{}
	err = configlatest.ReadYAML(configBytes, config)
	if err != nil {
		glog.V(5).Infof("%s error reading config: %v", api.PluginName, err)
		return nil, err
	}
	glog.V(5).Infof("%s config is: %v", api.PluginName, config)
	return config, nil
}

func (a *clusterResourceOverridePlugin) Validate() error {
	if err := validation.Validate(a.Config); err != nil {
		return err
	}
	if a.ProjectCache == nil {
		return fmt.Errorf("%s did not get a project cache", api.PluginName)
	}
	return nil
}

// TODO this will need to update when we have pod requests/limits
func (a *clusterResourceOverridePlugin) Admit(attr admission.Attributes) error {
	glog.V(6).Infof("%s admission controller is invoked", api.PluginName)
	if a.Config == nil || attr.GetResource() != kapi.Resource("pods") || attr.GetSubresource() != "" {
		return nil // not applicable
	}
	pod, ok := attr.GetObject().(*kapi.Pod)
	if !ok {
		return admission.NewForbidden(attr, fmt.Errorf("unexpected object: %#v", attr.GetObject()))
	}
	glog.V(5).Infof("%s is looking at creating pod %s in project %s", api.PluginName, pod.Name, attr.GetNamespace())

	// allow annotations on project to override
	if ns, err := a.ProjectCache.GetNamespace(attr.GetNamespace()); err != nil {
		glog.Warningf("%s got an error retrieving namespace: %v", api.PluginName, err)
		return admission.NewForbidden(attr, err) // this should not happen though
	} else {
		projectEnabledPlugin, exists := ns.Annotations[clusterResourceOverrideAnnotation]
		if exists && projectEnabledPlugin != "true" {
			glog.V(5).Infof("%s is disabled for project %s", api.PluginName, attr.GetNamespace())
			return nil // disabled for this project, do nothing
		}
	}

	// Reuse LimitRanger logic to apply limit/req defaults from the project. Ignore validation
	// errors, assume that LimitRanger will run after this plugin to validate.
	glog.V(5).Infof("%s: initial pod limits are: %#v", api.PluginName, pod.Spec.Containers[0].Resources)
	if err := a.LimitRanger.Admit(attr); err != nil {
		glog.V(5).Infof("%s: error from LimitRanger: %#v", api.PluginName, err)
	}
	glog.V(5).Infof("%s: pod limits after LimitRanger are: %#v", api.PluginName, pod.Spec.Containers[0].Resources)
	for _, container := range pod.Spec.Containers {
		resources := container.Resources
		memLimit := resources.Limits.Memory()
		// Although resource Quantity objects are used for both memory and CPU they're at a very
		// different scale. The inf.Dec number type is for decimal numbers down to 0.001,
		// stored as ints with a scale. Memory is measured in bytes, while CPU is in cores,
		// with 3 places of accuracy for millicores. In order to maintain 3 places of accuracy
		// for CPU there is some hocus-pocus with the "scale" component when multiplying
		// (inf.Dec does not have a multiplication method).
		if a.Config.LimitCPUToMemoryPercent != 0.0 {
			resources.Limits[kapi.ResourceCPU] = resource.Quantity{
				Amount: inf.NewDec(int64(float64(memLimit.Value())*cpuBaseScaleFactor*a.Config.LimitCPUToMemoryPercent/100.0), 3),
				Format: resources.Limits.Cpu().Format,
			}
		}
		if a.Config.CPURequestToLimitPercent != 0.0 {
			resources.Requests[kapi.ResourceCPU] = resource.Quantity{
				Amount: inf.NewDec(int64(float64(resources.Limits.Cpu().MilliValue())*a.Config.CPURequestToLimitPercent/100.0), 3),
				Format: resources.Requests.Cpu().Format,
			}
		}
		if a.Config.MemoryRequestToLimitPercent != 0.0 {
			resources.Requests[kapi.ResourceMemory] = resource.Quantity{
				Amount: inf.NewDec(int64(float64(resources.Limits.Memory().Value())*a.Config.MemoryRequestToLimitPercent/100.0), 0),
				Format: resources.Requests.Memory().Format,
			}
		}
	}
	glog.V(5).Infof("%s: pod limits after overrides are: %#v", api.PluginName, pod.Spec.Containers[0].Resources)
	return nil
}
