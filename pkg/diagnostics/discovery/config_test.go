package discovery // config

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/kubectl/resource"
	"github.com/kr/pretty"
	osclientcmd "github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/spf13/cobra"
	"testing"
)

func TestFactory(t *testing.T) {
	t.Fail()
	cmd := &cobra.Command{
		Use:   "diagnostics",
		Short: "Test command",
		Long:  "Long",
	}
	osFlags := cmd.PersistentFlags()
	factory := osclientcmd.New(osFlags)
	pretty.Println(factory)
	pretty.Println("\nAnd now, after getting config:\n")
	config, _ := factory.OpenShiftClientConfig.RawConfig()
	pretty.Println(factory, config)
	pretty.Println("\nAnd now, after creating a builder:\n")
	mapper, typer := factory.Object()
	builder := resource.NewBuilder(mapper, typer, factory.ClientMapperForCommand(nil)).
		ResourceTypeOrNameArgs(true, "projects").
		Latest()
	pretty.Println(factory, builder)
	pretty.Println("\nAnd now, after retrieving results:\n")
	list, _ := builder.Do().Infos()
	pretty.Println(factory, list)
	//pretty.Println("\nAnd now, after changing config:\n")
	//factory = osclientcmd.NewFactory(osFlags)
	//pretty.Println(factory)
}
