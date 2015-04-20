package cmd

import (
	"fmt"
	//"github.com/openshift/origin/pkg/cmd/cli/cmd"
	"github.com/openshift/origin/pkg/cmd/cli/config"
	"github.com/openshift/origin/pkg/cmd/server/start"
	"github.com/openshift/origin/pkg/cmd/templates"
	osclientcmd "github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/diagnostics/log"
	"github.com/openshift/origin/pkg/diagnostics/run"
	"github.com/openshift/origin/pkg/diagnostics/types"
	"github.com/spf13/cobra"
	"os"
)

const longAllDescription = `
OpenShift Diagnostics

This command helps you understand and troubleshoot OpenShift. It is
intended to be run from the same context as an OpenShift client or running
master / node in order to troubleshoot from the perspective of each.

    $ %s

If run without flags or subcommands, it will check for config files for
client, master, and node, and if found, use them for troubleshooting
those components. If master/node config files are not found, the tool
assumes they are not present and does diagnostics only as a client.

You may also specify config files explicitly with flags below, in which
case you will receive an error if they are not found or invalid.

    $ %[1]s --master-config=/etc/openshift/master.yaml

Subcommands may be used to scope the troubleshooting to a particular
component and are not limited to using config files; you can and should
use the same flags that are actually set on the command line for that
component to configure the diagnostic.

    $ %[1]s node --master=https://master.openshift.example.com:8443 --cert-dir=...

NOTE: This is an alpha version of diagnostics and will change significantly.
NOTE: Global flags (from the 'options' subcommand) are ignored here but
can be used with subcommands.
`

func NewCommand(name string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "This utility helps you understand and troubleshoot OpenShift v3.",
		Long:  fmt.Sprintf(longAllDescription, name),
	}
	diagFlags := types.NewFlags(cmd.PersistentFlags())
	addFlags(cmd, diagFlags)
	cmd.Flags().StringVar(&diagFlags.ClientConfigPath, config.OpenShiftConfigFlagName, "", "Path to the config file to use for client configuration.")
	cmd.Flags().StringVar(&diagFlags.MasterConfigPath, "master-config", "", "Path to the config file to use for master configuration.")
	cmd.Flags().StringVar(&diagFlags.NodeConfigPath, "node-config", "", "Path to the config file to use for node configuration.")

	/* There is some weirdness with diagnostics flag usage. The same flags
	   object is shared between the (implied) "all" and the client commands.
	   We actually use the client factory built in the "client" subcommand for
	   discovery in either case, and that adds flags to the subcommand which
	   need a target for putting results in; we do not want it to add flags
	   to the "all" command, but those flags have to exist somewhere for the
	   factory to set values and look them up later (even though on the "all"
	   command they will not exist; the only option is a client config file).
	   So the client flags object is reused for the "all" command.
	*/

	ccmd, factory := NewClientCommand(name+" client", diagFlags)
	cmd.AddCommand(ccmd)
	cmd.AddCommand(NewMasterCommand(name + " master"))
	cmd.AddCommand(NewNodeCommand(name+" node", diagFlags))
	cmd.AddCommand(NewOptionsCommand())

	cmd.Run = func(c *cobra.Command, args []string) {
		runInit(c, diagFlags)
		diagFlags.CanCheck[types.ClientTarget] = true
		diagFlags.CanCheck[types.MasterTarget] = true
		diagFlags.CanCheck[types.NodeTarget] = true
		run.Diagnose(diagFlags, factory)
	}
	return cmd
}

// Used in the command definition for common flags that we want to be visible in usage
func addFlags(cmd *cobra.Command, flags *types.Flags) {
	cmd.Flags().VarP(&flags.Diagnostics, "diagnostics", "d", `comma-separated list of diagnostic names to run, e.g. "systemd.AnalyzeLogs"`)
	cmd.Flags().IntVarP(&flags.LogLevel, "loglevel", "l", 3, "Level of output: 0: Error, 1: Warn, 2: Notice, 3: Info, 4: Debug")
	cmd.Flags().StringVarP(&flags.Format, "output", "o", "text", "Output format: text|json|yaml")
}

// Every command invocation needs to do the same things at the beginning...
func runInit(cmd *cobra.Command, diagFlags *types.Flags) {
	cmd.SetOutput(os.Stdout)                         // for output re usage / help
	diagFlags.OpenshiftFlags = cmd.PersistentFlags() // capture flags from *this* command in env
	log.SetLevel(diagFlags.LogLevel)
	log.SetLogFormat(diagFlags.Format) // note, ignore error; if format unknown, just do text
}

const longClientDescription = `
OpenShift Diagnostics

This command helps you understand and troubleshoot OpenShift as a user. It is
intended to be run from the same context as an OpenShift client
("openshift cli" or "osc") and with the same configuration options.

    $ %s
`

func NewClientCommand(fullName string, diagFlags *types.Flags) (*cobra.Command, *osclientcmd.Factory) {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "Troubleshoot using the OpenShift v3 client.",
		Long:  fmt.Sprintf(longClientDescription, fullName),
	}

	// Add some diagnostics flags to be shown separately from client flags
	addFlags(cmd, diagFlags)
	cmd.Flags().StringVarP(&diagFlags.OpenshiftPath, "openshift", "", "", "Path to 'openshift' binary")
	cmd.Flags().StringVarP(&diagFlags.OscPath, "osc", "", "", "Path to 'osc' client binary")

	factory := osclientcmd.New(cmd.PersistentFlags()) // side effect: add standard flags for openshift client
	// finally, set callback function for when this command is invoked:
	cmd.Run = func(c *cobra.Command, args []string) {
		runInit(c, diagFlags)
		diagFlags.CanCheck[types.ClientTarget] = true
		diagFlags.MustCheck = types.ClientTarget
		run.Diagnose(diagFlags, factory)
	}
	cmd.AddCommand(NewOptionsCommand())
	return cmd, factory
}

const longMasterDescription = `
OpenShift Diagnostics

This command helps you understand and troubleshoot a running OpenShift
master. It is intended to be run from the same context as the master
(where "openshift start" or "openshift start master" is run, possibly from
systemd or inside a container) and with the same configuration options.

    $ %s
`

func NewMasterCommand(fullName string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "master",
		Short: "Troubleshoot a running OpenShift v3 master.",
		Long:  fmt.Sprintf(longMasterDescription, fullName),
	}
	diagFlags := types.NewFlags(cmd.PersistentFlags())
	cmd.Run = func(c *cobra.Command, args []string) {
		runInit(c, diagFlags)
		diagFlags.CanCheck[types.MasterTarget] = true
		diagFlags.MustCheck = types.MasterTarget
		run.Diagnose(diagFlags, nil)
	}

	addFlags(cmd, diagFlags)
	diagFlags.MasterOptions = &start.MasterOptions{}
	cmd.Flags().StringVar(&diagFlags.MasterOptions.ConfigFile, "config", "", "Location of the master configuration file to run from. When running from a configuration file, all other command-line arguments are ignored.")
	diagFlags.MasterOptions.MasterArgs = start.MasterArgsAndFlags(cmd.Flags())

	cmd.AddCommand(NewOptionsCommand())
	return cmd
}

const longNodeDescription = `
OpenShift Diagnostics

This command helps you understand and troubleshoot a running OpenShift
node. It is intended to be run from the same context as the node
(where "openshift start" or "openshift start node" is run, possibly from
systemd or inside a container) and with the same configuration options.

    $ %s
`

func NewNodeCommand(fullName string, diagFlags *types.Flags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Troubleshoot an OpenShift v3 node.",
		Long:  fmt.Sprintf(longNodeDescription, fullName),
		Run: func(c *cobra.Command, args []string) {
			runInit(c, diagFlags)
			diagFlags.CanCheck[types.NodeTarget] = true
			diagFlags.MustCheck = types.NodeTarget
			run.Diagnose(diagFlags, nil)
		},
	}
	addFlags(cmd, diagFlags)
	cmd.AddCommand(NewOptionsCommand())
	return cmd
}

func NewOptionsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "options",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
	}

	templates.UseOptionsTemplates(cmd)

	return cmd
}
