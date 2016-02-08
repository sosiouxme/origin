package clusterresourceoverride

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/openshift/origin/pkg/cmd/server/api"
	projectcache "github.com/openshift/origin/pkg/project/cache"
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
limitCPUToMemoryRatio: 1.0
cpuRequestToLimitRatio: 0.1
memoryRequestToLimitRatio: 0.25
`
	invalidConfig = `
apiVersion: v1
kind: ClusterResourceOverrideConfig
enabled: true
cpuRequestToLimitRatio: 2.0
`
	invalidConfig2 = `Enabled: true`
)

const (
	oneGiB  = 1024 * 1024 * 1024
	oneCore = 1000 // milliCores, scaled by 3
)

var (
	configMeta = unversioned.TypeMeta{Kind: "ClusterResourceOverrideConfig", APIVersion: "v1"}
	//configMeta   = unversioned.TypeMeta{}
	targetConfig = api.ClusterResourceOverrideConfig{
		TypeMeta:                  configMeta,
		Enabled:                   true,
		LimitCPUToMemoryRatio:     1.0,
		CPURequestToLimitRatio:    0.1,
		MemoryRequestToLimitRatio: 0.25,
	}
)

func TestConfigReader(t *testing.T) {
	initial := testConfig(true, 0.1, 0.2, 0.3)
	if config, err := json.Marshal(initial); err != nil {
		t.Errorf("json.Marshal: config serialize failed: %v", err)
	} else if returned, readerr := ReadConfig(bytes.NewReader(config)); readerr != nil {
		t.Errorf("ReadConfig: config deserialize failed: %v", readerr)
	} else if returned != initial {
		t.Errorf("ReadConfig: expected %v, got %v", initial, returned)
	}

	if _, err := ReadConfig(bytes.NewReader([]byte("asdfasdfasdF"))); err == nil {
		t.Errorf("should have choked on broken config")
	}
	if config, err := ReadConfig(bytes.NewReader([]byte(yamlConfig))); err != nil {
		t.Errorf("should have been able to deserialize yaml: %v", err)
	} else if config != targetConfig {
		t.Errorf("target for yamlConfig: was %v, should have been %v", config, targetConfig)
	}
	if config, err := ReadConfig(bytes.NewReader([]byte(invalidConfig))); err != nil {
		t.Errorf("ReadConfig invalidConfig: config deserialize failed: %v", err)
	} else if err2 := Validate(config); err2 == nil {
		t.Errorf("should have choked on out-of-bounds ratio")
	}
	if config, err := ReadConfig(bytes.NewReader([]byte(invalidConfig2))); err != nil {
		t.Errorf("ReadConfig invalidConfig2: config deserialize failed: %v", err)
	} else if err2 := Validate(config); err2 == nil {
		t.Errorf("should have complained about no ratios being set")
	}

}

func TestLimitRequestAdmission(t *testing.T) {
	tests := []struct {
		name               string
		config             api.ClusterResourceOverrideConfig
		object             runtime.Object
		expectedMemRequest resource.Quantity
		expectedCpuLimit   resource.Quantity
		expectedCpuRequest resource.Quantity
		namespace          *kapi.Namespace
	}{
		{
			name:               "this thing even runs",
			config:             testConfig(true, 1.0, 0.5, 0.5),
			object:             testPod("0", "0", "0", "0"),
			expectedMemRequest: resource.MustParse("0"),
			expectedCpuLimit:   resource.MustParse("0"),
			expectedCpuRequest: resource.MustParse("0"),
			namespace:          fakeNamespace(true),
		},
		{
			name:               "all values are adjusted",
			config:             testConfig(true, 1.0, 0.5, 0.5),
			object:             testPod("1Gi", "0", "2000m", "0"),
			expectedMemRequest: resource.MustParse("512Mi"),
			expectedCpuLimit:   resource.MustParse("1"),
			expectedCpuRequest: resource.MustParse("500m"),
			namespace:          fakeNamespace(true),
		},
		{
			name:               "just requests are adjusted",
			config:             testConfig(true, 0.0, 0.5, 0.5),
			object:             testPod("10Mi", "0", "50m", "0"),
			expectedMemRequest: resource.MustParse("5Mi"),
			expectedCpuLimit:   resource.MustParse("50m"),
			expectedCpuRequest: resource.MustParse("25m"),
			namespace:          fakeNamespace(true),
		},
		{
			name:               "project annotation disables overrides",
			config:             testConfig(true, 0.0, 0.5, 0.5),
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

func testConfig(enabled bool, lc2mr float64, cr2lr float64, mr2lr float64) api.ClusterResourceOverrideConfig {
	return api.ClusterResourceOverrideConfig{
		TypeMeta:                  configMeta,
		Enabled:                   enabled,
		LimitCPUToMemoryRatio:     lc2mr,
		CPURequestToLimitRatio:    cr2lr,
		MemoryRequestToLimitRatio: mr2lr,
	}
}
