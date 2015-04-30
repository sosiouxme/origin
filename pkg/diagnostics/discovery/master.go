package discovery

import (
	mconfigapilatest "github.com/openshift/origin/pkg/cmd/server/api/latest"
	"github.com/openshift/origin/pkg/cmd/server/start"
	"github.com/openshift/origin/pkg/diagnostics/log"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

const StandardMasterConfPath string = "/etc/openshift/master.yaml"

func masterDiscovery(env *types.Environment, options *start.MasterOptions) {
	// first, determine if we even have a master config
	if options.ConfigFile != "" { // specified master conf, it has to load or we choke
		options.MasterArgs = start.NewDefaultMasterArgs() // ignore other flags
		if tryMasterConfig(env, options, true) {
			env.WillCheck[types.MasterTarget] = true
		}
	} else { // user did not indicate config file
		log.Debug("discMCnofile", "No master config file specified")
		if env.Flags.MustCheck != types.MasterTarget {
			// general command, user couldn't indicate server flags;
			// look for master config in standard location(s)
			tryStandardMasterConfig(env, options) // or give up.
		} else { // assume user provided flags like actual master.
			tryMasterConfig(env, options, true)
		}
	}
	if !env.WillCheck[types.MasterTarget] {
		log.Notice("discMCnone", "No master config found; master diagnostics will not be performed.")
	}
}

func tryMasterConfig(env *types.Environment, options *start.MasterOptions, errOnFail bool) bool {
	logOnFail := log.Debugf
	if errOnFail {
		logOnFail = log.Errorf
	}
	if err := options.Complete(); err != nil {
		logOnFail("discMCstart", "Could not read master config options: (%T) %[1]v", err)
		return false
	} else if err = options.Validate([]string{}); err != nil {
		logOnFail("discMCstart", "Could not read master config options: (%T) %[1]v", err)
		return false
	}
	var err error
	if path := options.ConfigFile; path != "" {
		log.Debugf("discMCfile", "Looking for master config file at '%s'", path)
		if env.MasterConfig, err = mconfigapilatest.ReadAndResolveMasterConfig(path); err != nil {
			logOnFail("discMCfail", "Could not read master config file '%s':\n(%T) %[2]v", path, err)
			return false
		}
		log.Infof("discMCfound", "Found a master config file:\n%[1]s", path)
	} else {
		if env.MasterConfig, err = options.MasterArgs.BuildSerializeableMasterConfig(); err != nil {
			logOnFail("discMCopts", "Could not build a master config from flags:\n(%T) %[1]v", err)
			return false
		}
		log.Infof("discMCfound", "No master config file, using any flags for configuration.")
	}
	if env.MasterConfig != nil {
		env.MasterOptions = options
		return true
	}
	return false
}

func tryStandardMasterConfig(env *types.Environment, options *start.MasterOptions) (worked bool) {
	log.Debug("discMCnoflags", "No master config flags specified, will try standard config location")
	options.ConfigFile = StandardMasterConfPath
	options.MasterArgs = start.NewDefaultMasterArgs()
	if tryMasterConfig(env, options, false) {
		log.Debug("discMCdefault", "Using master config file at "+StandardMasterConfPath)
		env.WillCheck[types.MasterTarget] = true
		return true
	} else { // otherwise, we just don't do master diagnostics
		log.Debugf("discMCnone", "Not using master config file at "+StandardMasterConfPath+" - will not do master diagnostics.")
	}
	return false
}
