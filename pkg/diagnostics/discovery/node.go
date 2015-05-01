package discovery

import (
	//"github.com/kr/pretty"
	mconfigapilatest "github.com/openshift/origin/pkg/cmd/server/api/latest"
	"github.com/openshift/origin/pkg/cmd/server/start"
	"github.com/openshift/origin/pkg/diagnostics/log"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

const StandardNodeConfPath string = "/etc/openshift/node/node-config.yaml"

func nodeDiscovery(env *types.Environment, options *start.NodeOptions) {
	// first, determine if we even have a node config
	if options.ConfigFile != "" { // specified node conf, it has to load or we choke
		options.NodeArgs = start.NewDefaultNodeArgs() // ignore other flags
		if tryNodeConfig(env, options, true) {
			env.WillCheck[types.NodeTarget] = true
		}
	} else { // user did not indicate config file
		log.Debug("discNCnofile", "No node config file specified")
		if env.Flags.MustCheck != types.NodeTarget {
			// general command, user couldn't indicate server flags;
			// look for node config in standard location(s)
			tryStandardNodeConfig(env, options) // or give up.
		} else { // assume user provided flags like actual node.
			tryNodeConfig(env, options, true)
		}
	}
	if !env.WillCheck[types.NodeTarget] {
		log.Notice("discNCnone", "No node config found; node diagnostics will not be performed.")
	}
}

func tryNodeConfig(env *types.Environment, options *start.NodeOptions, errOnFail bool) bool {
	//pretty.Println("nodeconfig options are:", options)
	logOnFail := log.Debugf
	if errOnFail {
		logOnFail = log.Errorf
	}
	if err := options.Complete(); err != nil {
		logOnFail("discNCstart", "Could not read node config options: (%T) %[1]v", err)
		return false
	} else if err = options.Validate([]string{}); err != nil {
		logOnFail("discNCstart", "Could not read node config options: (%T) %[1]v", err)
		return false
	}
	var err error
	if path := options.ConfigFile; path != "" {
		log.Debugf("discNCfile", "Looking for node config file at '%s'", path)
		if env.NodeConfig, err = mconfigapilatest.ReadAndResolveNodeConfig(path); err != nil {
			logOnFail("discNCfail", "Could not read node config file '%s':\n(%T) %[2]v", path, err)
			return false
		}
		log.Infof("discNCfound", "Found a node config file:\n%[1]s", path)
	} else {
		if env.NodeConfig, err = options.NodeArgs.BuildSerializeableNodeConfig(); err != nil {
			logOnFail("discNCopts", "Could not build a node config from flags:\n(%T) %[1]v", err)
			return false
		}
		log.Infof("discNCfound", "No node config file, using any flags for configuration.")
	}
	if env.NodeConfig != nil {
		env.NodeOptions = options
		return true
	}
	return false
}

func tryStandardNodeConfig(env *types.Environment, options *start.NodeOptions) (worked bool) {
	log.Debug("discNCnoflags", "No node config flags specified, will try standard config location")
	options.ConfigFile = StandardNodeConfPath
	options.NodeArgs = start.NewDefaultNodeArgs()
	if tryNodeConfig(env, options, false) {
		log.Debug("discNCdefault", "Using node config file at "+StandardNodeConfPath)
		env.WillCheck[types.NodeTarget] = true
		return true
	} else { // otherwise, we just don't do node diagnostics
		log.Debugf("discNCnone", "Not using node config file at "+StandardNodeConfPath+" - will not do node diagnostics.")
	}
	return false
}
