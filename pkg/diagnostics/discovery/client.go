package discovery // client

import (
	"fmt"
	"github.com/openshift/origin/pkg/diagnostics/log"
	"github.com/openshift/origin/pkg/diagnostics/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ----------------------------------------------------------
// Look for 'osc' and 'openshift' executables
func clientDiscovery(env *types.Environment) (err error) {
	log.Debug("discSearchExec", "Searching for executables in path:\n  "+strings.Join(filepath.SplitList(os.Getenv("PATH")), "\n  ")) //TODO for non-Linux OS
	env.OscPath = findExecAndLog("osc", env, env.Flags.OscPath)
	if env.OscPath != "" {
		env.OscVersion, err = getExecVersion(env.OscPath)
	}
	env.OpenshiftPath = findExecAndLog("openshift", env, env.Flags.OpenshiftPath)
	if env.OpenshiftPath != "" {
		env.OpenshiftVersion, err = getExecVersion(env.OpenshiftPath)
	}
	if env.OpenshiftVersion.NonZero() && env.OscVersion.NonZero() && !env.OpenshiftVersion.Eq(env.OscVersion) {
		log.Warnm("discVersionMM", log.Msg{"osV": env.OpenshiftVersion.GoString(), "oscV": env.OscVersion.GoString(),
			"text": fmt.Sprintf("'openshift' version %#v does not match 'osc' version %#v; update or remove the lower version", env.OpenshiftVersion, env.OscVersion)})
	}
	return err
}

// ----------------------------------------------------------
// Look for a specific executable and log what happens
func findExecAndLog(cmd string, env *types.Environment, pathflag string) string {
	if pathflag != "" { // look for it where the user said it would be
		if filepath.Base(pathflag) != cmd {
			log.Errorm("discExecFlag", log.Msg{"command": cmd, "path": pathflag, "tmpl": `
You specified that '{{.command}}' should be found at:
  {{.path}}
  but that file has the wrong name. The file name determines available functionality and must match.`})
		} else if _, err := exec.LookPath(pathflag); err == nil {
			log.Infom("discExecFound", log.Msg{"command": cmd, "path": pathflag,
				"tmpl": "Specified '{{.command}}' is executable at {{.path}}"})
			return pathflag
		} else if _, err := os.Stat(pathflag); os.IsNotExist(err) {
			log.Errorm("discExecNoExist", log.Msg{"command": cmd, "path": pathflag,
				"tmpl": "You specified that '{{.command}}' should be at {{.path}}\nbut that file does not exist."})
		} else {
			log.Errorm("discExecNot", log.Msg{"command": cmd, "path": pathflag,
				"tmpl": "You specified that '{{.command}}' should be at {{.path}}\nbut that file is not executable."})
		}
	} else { // look for it in the path
		path := findExecFor(cmd)
		if path == "" {
			log.Warnm("discExecNoPath", log.Msg{"command": cmd, "tmpl": "No '{{.command}}' executable was found in your path"})
		} else {
			log.Infom("discExecFound", log.Msg{"command": cmd, "path": path, "tmpl": "Found '{{.command}}' at {{.path}}"})
			return path
		}
	}
	return ""
}

// ----------------------------------------------------------
// Look in the path for an executable
func findExecFor(cmd string) string {
	path, err := exec.LookPath(cmd)
	if err == nil {
		return path
	}
	if runtime.GOOS == "windows" {
		path, err = exec.LookPath(cmd + ".exe")
		if err == nil {
			return path
		}
	}
	return ""
}

// ----------------------------------------------------------
// Invoke executable's "version" command to determine version
func getExecVersion(path string) (version types.Version, err error) {
	cmd := exec.Command(path, "version")
	var out []byte
	out, err = cmd.CombinedOutput()
	if err == nil {
		var name string
		var x, y, z int
		if scanned, err := fmt.Sscanf(string(out), "%s v%d.%d.%d", &name, &x, &y, &z); scanned > 1 {
			version = types.Version{x, y, z}
			log.Infom("discVersion", log.Msg{"tmpl": "version of {{.command}} is {{.version}}", "command": name, "version": version.GoString()})
		} else {
			log.Errorf("discVersErr", `
Expected version output from '%s version'
Could not parse output received:
%v
Error was: %#v`, path, string(out), err)
		}
	} else {
		switch err.(type) {
		case *exec.Error:
			log.Errorf("discVersErr", "error in executing '%v version': %v", path, err)
		case *exec.ExitError:
			log.Errorf("discVersErr", `
Executed '%v version' which exited with an error code.
This version is likely old or broken.
Error was '%v';
Output was:
%v`, path, err.Error(), log.LimitLines(string(out), 5))
		default:
			log.Errorf("discVersErr", "executed '%v version' but an error occurred:\n%v\nOutput was:\n%v", path, err, string(out))
		}
	}
	return version, err
}
