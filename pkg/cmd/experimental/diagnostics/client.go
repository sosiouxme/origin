package diagnostics

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	kclient "github.com/GoogleCloudPlatform/kubernetes/pkg/client"
	kclientcmdapi "github.com/GoogleCloudPlatform/kubernetes/pkg/client/clientcmd/api"
	kcmdutil "github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd/util"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"

	diagnosticflags "github.com/openshift/origin/pkg/cmd/experimental/diagnostics/options"
	osclientcmd "github.com/openshift/origin/pkg/cmd/util/clientcmd"
	clientdiagnostics "github.com/openshift/origin/pkg/diagnostics/client"
	"github.com/openshift/origin/pkg/diagnostics/log"
	diagnostictypes "github.com/openshift/origin/pkg/diagnostics/types"
)

var (
	AvailableClientDiagnostics = util.NewStringSet("ConfigContexts", "NodeDefinitions")
)

// user options for openshift-diagnostics client command
type ClientDiagnosticsOptions struct {
	RequestedDiagnostics util.StringList

	KubeClient *kclient.Client
	KubeConfig *kclientcmdapi.Config

	LogOptions *log.LoggerOptions
	Logger     *log.Logger
}

func (o *ClientDiagnosticsOptions) Complete() error {
	var err error
	o.Logger, err = o.LogOptions.NewLogger()
	if err != nil {
		return err
	}

	return nil
}

func (o ClientDiagnosticsOptions) RunDiagnostics() (bool, error, int, int) {
	diagnostics := map[string]diagnostictypes.Diagnostic{}
	for _, diagnosticName := range o.RequestedDiagnostics {
		switch diagnosticName {
		case "ConfigContexts":
			for contextName, _ := range o.KubeConfig.Contexts {
				diagnostics[diagnosticName+"["+contextName+"]"] = clientdiagnostics.ConfigContext{o.KubeConfig, contextName, o.Logger}
			}

		case "NodeDefinitions":
			diagnostics[diagnosticName] = clientdiagnostics.NodeDefinition{o.KubeClient, o.Logger}

		default:
			return true, fmt.Errorf("unknown diagnostic: %v", diagnosticName), 0, 1
		}
	}

	warnCount := 0
	errorCount := 0
	for name, diagnostic := range diagnostics {
		if canRun, reason := diagnostic.CanRun(); !canRun {
			if reason == nil {
				o.Logger.Noticet("diagSkip", "Skipping diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}",
					log.Hash{"area": "client", "name": name, "diag": diagnostic.Description()})
			} else {
				o.Logger.Noticet("diagSkip", "Skipping diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}\nBecause: {{.reason}}",
					log.Hash{"area": "client", "name": name, "diag": diagnostic.Description(), "reason": reason.Error()})
			}
			continue
		}

		o.Logger.Noticet("diagRun", "Running diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}",
			log.Hash{"area": "client", "name": name, "diag": diagnostic.Description()})
		r := diagnostic.Check()
		for _, entry := range r.Logs() {
			o.Logger.LogEntry(entry)
		}
		warnCount += len(r.Warnings())
		errorCount += len(r.Errors())
	}

	return errorCount > 0, nil, warnCount, errorCount
}
