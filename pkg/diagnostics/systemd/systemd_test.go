package systemd

import (
	"fmt"
	"github.com/openshift/origin/pkg/cmd/server/start"
	imagetemplate "github.com/openshift/origin/pkg/cmd/util/variable"
	"testing"
)

func TestUnitLogStartMatch(t *testing.T) {
	for _, unitSpec := range unitLogSpecs {
		switch unitSpec.Name {
		case "openshift-master":
			if !unitSpec.StartMatch.MatchString(fmt.Sprintf(start.MasterStartupMessage, "127.0.0.1", "3.0.0")) {
				t.Errorf("openshift-master StartMatch regex '%v' no longer matches master startup message '%s'", unitSpec.StartMatch, start.MasterStartupMessage)
			}
			break
		case "openshift-node":
			if !unitSpec.StartMatch.MatchString(fmt.Sprintf(start.NodeStartupMessage, "host.example.com")) {
				t.Errorf("openshift-node StartMatch regex '%v' no longer matches node startup message '%s'", unitSpec.StartMatch, start.NodeStartupMessage)
			}
			break
		}
	}
}

func TestUnitLogMatchers(t *testing.T) {
	if matcher := findLogMatcher("openshift-master", "DS2010", t); matcher != nil {
		_, err := (&imagetemplate.ImageTemplate{Format: "${completely bogus}"}).Expand("any")
		if msg := fmt.Sprintf(imagetemplate.BrokenImageFormat, "any", err); !matcher.Regexp.MatchString(msg) {
			t.Errorf("DS2010 regex '%v' no longer matches message '%s'", matcher.Regexp, msg)
		}
	}
}

func findLogMatcher(unit string, id string, t *testing.T) *logMatcher {
	var found *logMatcher
loop:
	for _, unitSpec := range unitLogSpecs {
		if unitSpec.Name == unit {
			for _, matcher := range unitSpec.LogMatchers {
				if matcher.Id == id {
					found = &matcher
					break loop
				}
			}
		}
	}
	if found == nil {
		t.Errorf("Could not find %s unit %s matcher", unit, id)
	}
	return found
}
