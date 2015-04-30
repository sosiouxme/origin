package types

import (
	"github.com/openshift/origin/pkg/cmd/server/start"
	flag "github.com/spf13/pflag"
	"strings"
)

type Flags struct {
	Diagnostics    List // named diagnostics to run
	OpenshiftPath  string
	OscPath        string
	LogLevel       int
	Format         string          // format of output - text/json/yaml
	OpenshiftFlags *flag.FlagSet   // flags that were set on the command run
	CanCheck       map[Target]bool // components to diagnose - master/node/client
	MustCheck      Target
	// for diagnosing "all", enable specifying config file locations
	ClientConfigPath string
	MasterConfigPath string
	NodeConfigPath   string
	// the master has its own container for flags/options
	MasterOptions *start.MasterOptions
	// the node has its own container for flags/options
	NodeOptions *start.NodeOptions
}

func NewFlags(flags *flag.FlagSet) *Flags {
	return &Flags{
		OpenshiftFlags: flags,
		Diagnostics:    make(List, 0),
		CanCheck:       make(map[Target]bool),
	}
}

type List []string

func (l *List) Set(arg string) error {
	*l = strings.Split(arg, ",")
	return nil
}
func (l *List) Type() string {
	return "list"
}
func (l *List) String() string {
	return strings.Join(*l, ",")
}

type Target string

const ClientTarget Target = "client"
const MasterTarget Target = "master"
const NodeTarget Target = "node"
