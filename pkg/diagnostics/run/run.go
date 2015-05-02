package run

import (
	osclientcmd "github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/diagnostics/client"
	"github.com/openshift/origin/pkg/diagnostics/discovery"
	"github.com/openshift/origin/pkg/diagnostics/log"
	"github.com/openshift/origin/pkg/diagnostics/systemd"
	"github.com/openshift/origin/pkg/diagnostics/types"
	"os"
	"strings"
)

func Diagnose(fl *types.Flags, f *osclientcmd.Factory) {
	if env, ok := discovery.Run(fl, f); ok { // discovery result can veto continuing
		allDiags := make(map[string]map[string]types.Diagnostic)
		for area, _ := range env.WillCheck {
			switch area {
			case types.ClientTarget:
				allDiags["client"] = client.Diagnostics
			case types.MasterTarget, types.NodeTarget:
				allDiags["systemd"] = systemd.Diagnostics
			}
		}
		if list := env.Flags.Diagnostics; len(list) > 0 {
			// just run a specific (set of) diagnostic(s)
			for _, arg := range list {
				parts := strings.SplitN(arg, ".", 2)
				if len(parts) < 2 {
					log.Noticef("noDiag", `There is no such diagnostic "%s"`, arg)
					continue
				}
				area, name := parts[0], parts[1]
				if diagnostics, exists := allDiags[area]; !exists {
					log.Noticef("noDiag", `There is no such diagnostic "%s"`, arg)
				} else if diag, exists := diagnostics[name]; !exists {
					log.Noticef("noDiag", `There is no such diagnostic "%s"`, arg)
				} else {
					RunDiagnostic(area, name, diag, env)
				}
			}
		} else {
			// TODO: run all of these in parallel but ensure sane output
			for area, diagnostics := range allDiags {
				for name, diag := range diagnostics {
					RunDiagnostic(area, name, diag, env)
				}
			}
		}
	}
	log.Summary()
	log.Finish()
	if log.ErrorsSeen() {
		os.Exit(255)
	}
}

func RunDiagnostic(area string, name string, diag types.Diagnostic, env *types.Environment) {
	defer func() {
		// recover from diagnostics that panic so others can still run
		if r := recover(); r != nil {
			log.Errorf("diagPanic", "Diagnostic '%s' crashed; this is usually a bug in either diagnostics or OpenShift. Stack trace:\n%+v", name, r)
		}
	}()
	if diag.Condition != nil {
		if skip, reason := diag.Condition(env); skip {
			if reason == "" {
				log.Noticem("diagSkip", log.Msg{"area": area, "name": name, "diag": diag.Description,
					"tmpl": "Skipping diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}"})
			} else {
				log.Noticem("diagSkip", log.Msg{"area": area, "name": name, "diag": diag.Description, "reason": reason,
					"tmpl": "Skipping diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}\nBecause: {{.reason}}"})
			}
			return
		}
	}
	log.Noticem("diagRun", log.Msg{"area": area, "name": name, "diag": diag.Description,
		"tmpl": "Running diagnostic: {{.area}}.{{.name}}\nDescription: {{.diag}}"})
	diag.Run(env)
}
