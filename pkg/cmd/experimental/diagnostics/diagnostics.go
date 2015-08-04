package diagnostics

import (
	"fmt"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"io"
	"os"

	kcmdutil "github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/cmd/util"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	kutilerrors "github.com/GoogleCloudPlatform/kubernetes/pkg/util/errors"
	"github.com/openshift/origin/pkg/cmd/cli/config"
	"github.com/openshift/origin/pkg/cmd/flagtypes"
	osclientcmd "github.com/openshift/origin/pkg/cmd/util/clientcmd"

	"github.com/openshift/origin/pkg/cmd/experimental/diagnostics/options"
	"github.com/openshift/origin/pkg/diagnostics/log"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

var (
	AvailableDiagnostics = util.NewStringSet()
)

func init() {
	AvailableDiagnostics.Insert(AvailableClientDiagnostics.List()...)
	AvailableDiagnostics.Insert(AvailableClusterDiagnostics.List()...)
	AvailableDiagnostics.Insert(AvailableHostDiagnostics.List()...)
}

type DiagnosticsOptions struct {
	RequestedDiagnostics util.StringList

	MasterConfigLocation string
	NodeConfigLocation   string
	ClientClusterContext string
	IsHost               bool

	ClientFlags *flag.FlagSet
	Factory     *osclientcmd.Factory

	LogOptions *log.LoggerOptions
	Logger     *log.Logger
}

const longDescription = `
OpenShift Diagnostics

This command helps you understand and troubleshoot OpenShift. It is
intended to be run from the same context as an OpenShift client or running
master / node in order to troubleshoot from the perspective of each.

    $ %[1]s

If run without flags or subcommands, it will check for config files for
client, master, and node, and if found, use them for troubleshooting
those components. If master/node config files are not found, the tool
assumes they are not present and does diagnostics only as a client.

You may also specify config files explicitly with flags below, in which
case you will receive an error if they are invalid or not found.

    $ %[1]s --master-config=/etc/openshift/master/master-config.yaml

Subcommands may be used to scope the troubleshooting to a particular
component and are not limited to using config files; you can and should
use the same flags that are actually set on the command line for that
component to configure the diagnostic.

    $ %[1]s node --hostname='node.example.com' --kubeconfig=...

NOTE: This is a beta version of diagnostics and may evolve significantly.
`

func NewCommandDiagnostics(name string, fullName string, out io.Writer) *cobra.Command {
	o := &DiagnosticsOptions{
		RequestedDiagnostics: AvailableDiagnostics.List(),
		LogOptions:           &log.LoggerOptions{Out: out},
	}

	cmd := &cobra.Command{
		Use:   name,
		Short: "This utility helps you understand and troubleshoot OpenShift v3.",
		Long:  fmt.Sprintf(longDescription, fullName),
		Run: func(c *cobra.Command, args []string) {
			kcmdutil.CheckErr(o.Complete())

			failed, err, warnCount, errorCount := o.RunDiagnostics()
			o.Logger.Summary(warnCount, errorCount)
			o.Logger.Finish()

			kcmdutil.CheckErr(err)
			if failed {
				os.Exit(255)
			}

		},
	}
	cmd.SetOutput(out) // for output re: usage / help

	o.ClientFlags = flag.NewFlagSet("client", flag.ContinueOnError) // hide the extensive set of client flags
	o.Factory = osclientcmd.New(o.ClientFlags)                      // that would otherwise be added to this command
	cmd.Flags().AddFlag(o.ClientFlags.Lookup(config.OpenShiftConfigFlagName))
	cmd.Flags().AddFlag(o.ClientFlags.Lookup("context")) // TODO: find k8s constant
	cmd.Flags().StringVar(&o.ClientClusterContext, options.FlagClusterContextName, "", "client context to use for cluster administrator")
	cmd.Flags().StringVar(&o.MasterConfigLocation, options.FlagMasterConfigName, "", "path to master config file (implies --host)")
	cmd.Flags().StringVar(&o.NodeConfigLocation, options.FlagNodeConfigName, "", "path to node config file (implies --host)")
	cmd.Flags().BoolVar(&o.IsHost, options.FlagIsHostName, false, "look for systemd and journald units even without master/node config")
	flagtypes.GLog(cmd.Flags())
	options.BindLoggerOptionFlags(cmd.Flags(), o.LogOptions, options.RecommendedLoggerOptionFlags())
	options.BindDiagnosticFlag(cmd.Flags(), &o.RequestedDiagnostics, options.NewRecommendedDiagnosticFlag())

	return cmd
}

func (o *DiagnosticsOptions) Complete() error {
	var err error
	o.Logger, err = o.LogOptions.NewLogger()
	if err != nil {
		return err
	}

	return nil
}

func (o DiagnosticsOptions) RunDiagnostics() (bool, error, int, int) {
	failed := false
	warnings := []error{}
	errors := []error{}
	diagnostics := map[string][]types.Diagnostic{}

	func() { // don't trust discovery/build of diagnostics; wrap panic nicely in case of developer error
		defer func() {
			if r := recover(); r != nil {
				failed = true
				errors = append(errors, fmt.Errorf("While building the diagnostics, a panic was encountered.\nThis is a bug in diagnostics. Stack trace follows : \n%v", r))
			}
		}()
		detected, detectWarnings, detectErrors := o.detectClientConfig() // may log and return problems
		for _, warn := range detectWarnings {
			warnings = append(warnings, warn)
		}
		for _, err := range detectErrors {
			errors = append(errors, err)
		}
		if !detected { // there just plain isn't any client config file available
			o.Logger.Notice("discNoClientConf", "No client configuration specified; skipping client and cluster diagnostics.")
		} else if rawConfig, err := o.buildRawConfig(); rawConfig == nil { // client config is totally broken - won't parse etc (problems may have been detected and logged)
			o.Logger.Errorf("discBrokenClientConf", "Client configuration failed to load; skipping client and cluster diagnostics due to error: {{.error}}", log.Hash{"error": err.Error()})
			errors = append(errors, err)
		} else {
			if err != nil { // error encountered, proceed with caution
				o.Logger.Errorf("discClientConfErr", "Client configuration loading encountered an error, but proceeding anyway. Error was:\n{{.error}}", log.Hash{"error": err.Error()})
				errors = append(errors, err)
			}
			if clientDiags, ok, err := o.buildClientDiagnostics(rawConfig); ok {
				diagnostics["client"] = clientDiags
			} else if err != nil {
				failed = true
				errors = append(errors, err)
			}

			if clusterDiags, ok, err := o.buildClusterDiagnostics(rawConfig); ok {
				diagnostics["cluster"] = clusterDiags
			} else if err != nil {
				failed = true
				errors = append(errors, err)
			}
		}

		if hostDiags, ok, err := o.buildHostDiagnostics(); ok {
			diagnostics["host"] = hostDiags
		} else if err != nil {
			failed = true
			errors = append(errors, err)
		}
	}()

	if failed {
		return failed, kutilerrors.NewAggregate(errors), len(warnings), len(errors)
	}

	return o.Run(diagnostics)
}

func (o DiagnosticsOptions) Run(diagnostics map[string][]types.Diagnostic) (bool, error, int, int) {
	warnCount := 0
	errorCount := 0
	for area, areaDiagnostics := range diagnostics {
		for _, diagnostic := range areaDiagnostics {
			func() { // wrap diagnostic panic nicely in case of developer error
				defer func() {
					if r := recover(); r != nil {
						errorCount += 1
						o.Logger.Errort("diagPanic",
							"While running the {{.area}}.{{.name}} diagnostic, a panic was encountered.\nThis is a bug in diagnostics. Stack trace follows : \n{{.error}}",
							log.Hash{"area": area, "name": diagnostic.Name(), "error": fmt.Sprintf("%v", r)})
					}
				}()

				if canRun, reason := diagnostic.CanRun(); !canRun {
					if reason == nil {
						o.Logger.Noticet("diagSkip", "Skipping diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}",
							log.Hash{"area": area, "name": diagnostic.Name(), "diag": diagnostic.Description()})
					} else {
						o.Logger.Noticet("diagSkip", "Skipping diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}\nBecause: {{.reason}}",
							log.Hash{"area": area, "name": diagnostic.Name(), "diag": diagnostic.Description(), "reason": reason.Error()})
					}
					return
				}

				o.Logger.Noticet("diagRun", "Running diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}",
					log.Hash{"area": area, "name": diagnostic.Name(), "diag": diagnostic.Description()})
				r := diagnostic.Check()
				for _, entry := range r.Logs() {
					o.Logger.LogEntry(entry)
				}
				warnCount += len(r.Warnings())
				errorCount += len(r.Errors())
			}()
		}

	}
	return errorCount > 0, nil, warnCount, errorCount
}

// TODO move upstream
func intersection(s1 util.StringSet, s2 util.StringSet) util.StringSet {
	result := util.NewStringSet()
	for key := range s1 {
		if s2.Has(key) {
			result.Insert(key)
		}
	}
	return result
}
