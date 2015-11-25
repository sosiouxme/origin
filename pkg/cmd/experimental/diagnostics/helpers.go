package diagnostics

import (
	"fmt"

	"github.com/openshift/origin/pkg/diagnostics/log"
	"k8s.io/kubernetes/pkg/util/sets"
)

// determineRequestedDiagnostics determines which diagnostic the user wants to run
// based on the -d list and the available diagnostics.
// returns ok (can proceed), warnings, errors, diagnostics
func determineRequestedDiagnostics(available []string, requested []string, logger *log.Logger) (bool, []error, []error, []string) {
	ok := true
	warnings := []error{}
	errors := []error{}
	diagnostics := []string{}
	if len(requested) == 0 { // not specified, use the available list
		diagnostics = available
	} else if diagnostics = intersection(sets.NewString(requested...), sets.NewString(available...)).List(); len(diagnostics) == 0 {
		ok = false
		errors = append(errors, fmt.Errorf("No requested diagnostics available"))
		logger.Error("CED6001", log.EvalTemplate("CED6001", "None of the requested diagnostics are available:\n  {{.requested}}\nPlease try from the following:\n  {{.available}}",
			log.Hash{"requested": requested, "available": available}))
	} else if len(diagnostics) < len(requested) {
		ok = false
		errors = append(errors, fmt.Errorf("Not all requested diagnostics are available"))
		logger.Error("CED6002", log.EvalTemplate("CED6002", "Of the requested diagnostics:\n    {{.requested}}\nonly these are available:\n    {{.diagnostics}}\nThe list of all possible is:\n    {{.available}}",
			log.Hash{"requested": requested, "diagnostics": diagnostics, "available": available}))
	} // else it's a valid list.
	return ok, warnings, errors, diagnostics
}
