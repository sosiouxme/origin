package types

import (
	flag "github.com/spf13/pflag"
	"strings"
)

type Flags struct {
	Diagnostics    List
	LogLevel       int
	OpenshiftPath  string
	OscPath        string
	Format         string
	OpenshiftFlags *flag.FlagSet
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
