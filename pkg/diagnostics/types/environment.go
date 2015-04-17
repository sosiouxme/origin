package types

import (
	kclientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	mconfigapi "github.com/openshift/origin/pkg/cmd/server/api"
	mstart "github.com/openshift/origin/pkg/cmd/server/start"
	osclientcmd "github.com/openshift/origin/pkg/cmd/util/clientcmd"
)

// One env instance is created and filled in by discovery.
// Then it should be considered immutable while diagnostics use it.
type Environment struct {
	OS           string // "linux / windows / darwin" http://golang.org/pkg/runtime/#GOOS
	HasSystemd   bool
	HasBash      bool
	SystemdUnits map[string]SystemdUnit // list of relevant units present on system
	Flags        *Flags                 // command flags deposit results here; also has command flag objects
	WillCheck    map[Target]bool        // discover whether to diagnose master,node,client

	// outcome from looking for executables
	OscPath          string
	OscVersion       Version
	OpenshiftPath    string
	OpenshiftVersion Version

	// saved results from client discovery
	ClientConfigPath    string                          // first client config file found, if any
	ClientConfigRaw     *kclientcmdapi.Config           // available to analyze ^^
	OsConfig            *kclientcmdapi.Config           // actual merged client configuration
	FactoryForContext   map[string]*osclientcmd.Factory // one for each known context
	AccessForContext    map[string]*ContextAccess       // one for each context that has access to anything
	ClusterAdminFactory *osclientcmd.Factory            // factory we will use for cluster-admin access (could easily be nil)

	// saved results from master discovery
	MasterOptions *mstart.MasterOptions    // user-specified flags or config file
	MasterConfig  *mconfigapi.MasterConfig // actual config determined from flags/file
}

type ContextAccess struct {
	Projects     []string
	ClusterAdmin bool // has access to see stuff only cluster-admin should
}

func NewEnvironment(fl *Flags) *Environment {
	return &Environment{
		Flags:             fl,
		SystemdUnits:      make(map[string]SystemdUnit),
		WillCheck:         make(map[Target]bool),
		FactoryForContext: make(map[string]*osclientcmd.Factory),
		AccessForContext:  make(map[string]*ContextAccess),
	}
}

// helpful translator
func (env *Environment) DefaultFactory() *osclientcmd.Factory {
	if env.FactoryForContext != nil && env.OsConfig != nil { // no need to panic if missing...
		return env.FactoryForContext[env.OsConfig.CurrentContext]
	}
	return nil
}
