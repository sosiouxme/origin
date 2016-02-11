package api

import (
	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/runtime"

	configapi "github.com/openshift/origin/pkg/cmd/server/api"
	"github.com/openshift/origin/pkg/quota/admission/clusterresourceoverride/api/v1"
)

var SchemeGroupVersion = unversioned.GroupVersion{Group: "", Version: runtime.APIVersionInternal}

// Adds the list of known types to api.Scheme.
func AddToScheme(scheme *runtime.Scheme) {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&ClusterResourceOverrideConfig{},
	)
}

// availableVersions lists all known external versions for this group from most preferred to least preferred
var availableVersions = []unversioned.GroupVersion{v1.SchemeGroupVersion}

func init() {
	if err := enableVersions(availableVersions); err != nil {
		panic(err)
	}
}

// TODO: enableVersions should be centralized rather than spread in each API
// group.
// We can combine registered.RegisterVersions, registered.EnableVersions and
// registered.RegisterGroup once we have moved enableVersions there.
func enableVersions(externalVersions []unversioned.GroupVersion) error {
	addVersionsToScheme(externalVersions...)
	return nil
}

func addVersionsToScheme(externalVersions ...unversioned.GroupVersion) {
	// add the internal version to Scheme
	AddToScheme(configapi.Scheme)
	// add the enabled external versions to Scheme
	for _, v := range externalVersions {
		switch v {
		case v1.SchemeGroupVersion:
			v1.AddToScheme(configapi.Scheme)

		default:
			glog.Errorf("Version %s is not known, so it will not be added to the Scheme.", v)
			continue
		}
	}
}

func (obj *ClusterResourceOverrideConfig) GetObjectKind() unversioned.ObjectKind { return &obj.TypeMeta }
