package clusterresourceoverride

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	projectcache "github.com/openshift/origin/pkg/project/cache"
	"github.com/openshift/origin/pkg/quota/admission/clusterresourceoverride/api"
	"github.com/openshift/origin/pkg/quota/admission/clusterresourceoverride/api/validation"
	"k8s.io/kubernetes/pkg/admission"
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/auth/user"
	"k8s.io/kubernetes/pkg/client/cache"
	ktestclient "k8s.io/kubernetes/pkg/client/unversioned/testclient"
	"k8s.io/kubernetes/pkg/runtime"
)

const (
	yamlConfig = `
apiVersion: v1
kind: ClusterResourceOverrideConfig
enabled: true
limitCPUToMemoryPercent: 100
cpuRequestToLimitPercent: 10
memoryRequestToLimitPercent: 25
`
	invalidConfig = `
apiVersion: v1
kind: ClusterResourceOverrideConfig
enabled: true
cpuRequestToLimitPercent: 200
`
	invalidConfig2 = `Enabled: true`
)

var (
	configMeta = unversioned.TypeMeta{Kind: api.ConfigKind, APIVersion: "v1"}
	//configMeta   = unversioned.TypeMeta{}
	targetConfig = &api.ClusterResourceOverrideConfig{
		TypeMeta:                    configMeta,
		LimitCPUToMemoryPercent:     100,
		CPURequestToLimitPercent:    10,
		MemoryRequestToLimitPercent: 25,
	}
)

func TestConfigReader(t *testing.T) {
	if empty, err := ReadConfig(nil); err != nil {
		t.Errorf("error on processing nil config: %v", err)
	} else if empty != nil {
		t.Errorf("should have gotten nil reading a nil config: %v", err)
	}

	initial := testConfig(true, 10, 20, 30)
	if config, err := json.Marshal(initial); err != nil {
		t.Errorf("json.Marshal: config serialize failed: %v", err)
	} else if returned, readerr := ReadConfig(bytes.NewReader(config)); readerr != nil {
		t.Errorf("ReadConfig: config deserialize failed: %v", readerr)
	} else if *returned != *initial {
		t.Errorf("ReadConfig: expected %v, got %v", initial, returned)
	}

	if _, err := ReadConfig(bytes.NewReader([]byte("asdfasdfasdF"))); err == nil {
		t.Errorf("should have choked on broken config")
	}
	if config, err := ReadConfig(bytes.NewReader([]byte(yamlConfig))); err != nil {
		t.Errorf("should have been able to deserialize yaml: %v", err)
	} else if *config != *targetConfig {
		t.Errorf("target for yamlConfig: was %v, should have been %v", config, targetConfig)
	}
	if config, err := ReadConfig(bytes.NewReader([]byte(invalidConfig))); err != nil {
		t.Errorf("ReadConfig invalidConfig: config deserialize failed: %v", err)
	} else if err2 := validation.Validate(config); err2 == nil {
		t.Errorf("should have choked on out-of-bounds ratio")
	}
	if config, err := ReadConfig(bytes.NewReader([]byte(invalidConfig2))); err != nil {
		t.Errorf("ReadConfig invalidConfig2: config deserialize failed: %v", err)
	} else if err2 := validation.Validate(config); err2 == nil {
		t.Errorf("should have complained about no ratios being set")
	}

}

func TestLimitRequestAdmission(t *testing.T) {
	tests := []struct {
		name               string
		config             *api.ClusterResourceOverrideConfig
		object             runtime.Object
		expectedMemRequest resource.Quantity
		expectedCpuLimit   resource.Quantity
		expectedCpuRequest resource.Quantity
		namespace          *kapi.Namespace
	}{
		{
			name:               "this thing even runs",
			config:             testConfig(true, 100, 50, 50),
			object:             testPod("0", "0", "0", "0"),
			expectedMemRequest: resource.MustParse("0"),
			expectedCpuLimit:   resource.MustParse("0"),
			expectedCpuRequest: resource.MustParse("0"),
			namespace:          fakeNamespace(true),
		},
		{
			name:               "all values are adjusted",
			config:             testConfig(true, 100, 50, 50),
			object:             testPod("1Gi", "0", "2000m", "0"),
			expectedMemRequest: resource.MustParse("512Mi"),
			expectedCpuLimit:   resource.MustParse("1"),
			expectedCpuRequest: resource.MustParse("500m"),
			namespace:          fakeNamespace(true),
		},
		{
			name:               "just requests are adjusted",
			config:             testConfig(true, 0, 50, 50),
			object:             testPod("10Mi", "0", "50m", "0"),
			expectedMemRequest: resource.MustParse("5Mi"),
			expectedCpuLimit:   resource.MustParse("50m"),
			expectedCpuRequest: resource.MustParse("25m"),
			namespace:          fakeNamespace(true),
		},
		{
			name:               "project annotation disables overrides",
			config:             testConfig(true, 0, 50, 50),
			object:             testPod("10Mi", "0", "50m", "0"),
			expectedMemRequest: resource.MustParse("0"),
			expectedCpuLimit:   resource.MustParse("50m"),
			expectedCpuRequest: resource.MustParse("0"),
			namespace:          fakeNamespace(false),
		},
	}

	for _, test := range tests {
		config, _ := json.Marshal(test.config)
		c, err := newClusterResourceOverride(&ktestclient.Fake{}, bytes.NewReader(config))
		if err != nil {
			t.Errorf("%s: config de/serialize failed: %v", test.name, err)
		}
		c.(*clusterResourceOverridePlugin).SetProjectCache(fakeProjectCache(test.namespace))
		attrs := admission.NewAttributesRecord(test.object, unversioned.GroupKind{}, test.namespace.Name, "name", kapi.Resource("pods"), "", admission.Create, fakeUser())
		if err := c.Admit(attrs); err != nil {
			t.Errorf("%s: admission controller should not return error", test.name)
		}
		// if it's a pod, test that the resources are as expected
		pod, ok := test.object.(*kapi.Pod)
		if !ok {
			continue
		}
		resources := pod.Spec.Containers[0].Resources // only test one container
		if actual := resources.Requests[kapi.ResourceMemory]; test.expectedMemRequest.Cmp(actual) != 0 {
			t.Errorf("%s: memory requests do not match; %s should be %s", test.name, actual, test.expectedMemRequest)
		}
		if actual := resources.Requests[kapi.ResourceCPU]; test.expectedCpuRequest.Cmp(actual) != 0 {
			t.Errorf("%s: cpu requests do not match; %s should be %s", test.name, actual, test.expectedCpuRequest)
		}
		if actual := resources.Limits[kapi.ResourceCPU]; test.expectedCpuLimit.Cmp(actual) != 0 {
			t.Errorf("%s: cpu limits do not match; %s should be %s", test.name, actual, test.expectedCpuLimit)
		}
	}
}

func testPod(memLimit string, memRequest string, cpuLimit string, cpuRequest string) *kapi.Pod {
	return &kapi.Pod{
		Spec: kapi.PodSpec{
			Containers: []kapi.Container{
				{
					Resources: kapi.ResourceRequirements{
						Limits: kapi.ResourceList{
							kapi.ResourceCPU:    resource.MustParse(cpuLimit),
							kapi.ResourceMemory: resource.MustParse(memLimit),
						},
						Requests: kapi.ResourceList{
							kapi.ResourceCPU:    resource.MustParse(cpuRequest),
							kapi.ResourceMemory: resource.MustParse(memRequest),
						},
					},
				},
			},
		},
	}
}

func fakeUser() user.Info {
	return &user.DefaultInfo{
		Name: "testuser",
	}
}

var nsIndex = 0

func fakeNamespace(pluginEnabled bool) *kapi.Namespace {
	nsIndex++
	ns := &kapi.Namespace{
		ObjectMeta: kapi.ObjectMeta{
			Name:        fmt.Sprintf("fakeNS%d", nsIndex),
			Annotations: map[string]string{},
		},
	}
	if !pluginEnabled {
		ns.Annotations[clusterResourceOverrideAnnotation] = "false"
	}
	return ns
}

func fakeProjectCache(ns *kapi.Namespace) *projectcache.ProjectCache {
	store := projectcache.NewCacheStore(cache.MetaNamespaceKeyFunc)
	store.Add(ns)
	return projectcache.NewFake((&ktestclient.Fake{}).Namespaces(), store, "")
}

func testConfig(enabled bool, lc2mr float64, cr2lr float64, mr2lr float64) *api.ClusterResourceOverrideConfig {
	return &api.ClusterResourceOverrideConfig{
		TypeMeta:                    configMeta,
		LimitCPUToMemoryPercent:     lc2mr,
		CPURequestToLimitPercent:    cr2lr,
		MemoryRequestToLimitPercent: mr2lr,
	}
}
