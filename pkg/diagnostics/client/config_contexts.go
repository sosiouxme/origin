package client

import (
	"errors"
	"fmt"

	kapi "github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	kclientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"

	"github.com/openshift/origin/pkg/diagnostics/types"
)

const (
	currentContextMissing = `Your client config specifies a current context of '{{.context}}'
which is not defined; it is likely that a mistake was introduced while
manually editing your config. If this is a simple typo, you may be
able to fix it manually.
The OpenShift master creates a fresh config when it is started; it may be
useful to use this as a base if available.`

	currentContextSummary = `The current context from client config is '{{.context}}'
This will be used by default to contact your OpenShift server.
`
)

type ConfigContext struct {
	KubeConfig  *kclientcmdapi.Config
	ContextName string
}

func (d ConfigContext) Name() string {
	return fmt.Sprintf("ConfigContext[%s]", d.ContextName)
}

func (d ConfigContext) Description() string {
	return "Test that client config contexts have no undefined references"
}

func (d ConfigContext) CanRun() (bool, error) {
	if d.KubeConfig == nil {
		// TODO make prettier?
		return false, errors.New("There is no client config file")
	}

	if len(d.ContextName) == 0 {
		return false, errors.New("There is no current context")
	}

	return true, nil
}

func (d ConfigContext) Check() *types.DiagnosticResult {
	r := types.NewDiagnosticResult("ConfigContext")

	isDefaultContext := d.KubeConfig.CurrentContext == d.ContextName

	errorKey := "clientCfgError"
	unusableLine := fmt.Sprintf("The client config context '%s' is unusable", d.ContextName)
	if isDefaultContext {
		errorKey = "currentccError"
		unusableLine = fmt.Sprintf("The current client config context '%s' is unusable", d.ContextName)
	}

	context, exists := d.KubeConfig.Contexts[d.ContextName]
	if !exists {
		r.Errorf(errorKey, nil, "%s:\n Client config context '%s' is not defined.", unusableLine, d.ContextName)
		return r
	}

	clusterName := context.Cluster
	cluster, exists := d.KubeConfig.Clusters[clusterName]
	if !exists {
		r.Errorf(errorKey, nil, "%s:\n Client config context '%s' has a cluster '%s' which is not defined.", unusableLine, d.ContextName, clusterName)
		return r
	}
	authName := context.AuthInfo
	if _, exists := d.KubeConfig.AuthInfos[authName]; !exists {
		r.Errorf(errorKey, nil, "%s:\n Client config context '%s' has a user identity '%s' which is not defined.", unusableLine, d.ContextName, authName)
		return r
	}

	project := context.Namespace
	if project == "" {
		project = kapi.NamespaceDefault // OpenShift/k8s fills this in if missing
	}

	// TODO: actually send a request to see if can connect
	text := "For client config context '%s':\n The server URL is '%s'\nThe user authentication is '%s'\nThe current project is '%s'"
	if isDefaultContext {
		text = "The current client config context is '%s':\n The server URL is '%s'\nThe user authentication is '%s'\nThe current project is '%s'"
	}
	r.Infof("ccInfo", text, d.ContextName, cluster.Server, authName, project)
	return r
}
