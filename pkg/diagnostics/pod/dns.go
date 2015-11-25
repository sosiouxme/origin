package host

import (
	"errors"
	"fmt"
	"ioutil"
	"regexp"

	"github.com/miekg/dns"
	configapilatest "github.com/openshift/origin/pkg/cmd/server/api/latest"
	configvalidation "github.com/openshift/origin/pkg/cmd/server/api/validation"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

// PodCheckDns is a Diagnostic to check that DNS within a pod works as expected
type PodCheckDns struct {
}

const PodCheckDnsName = "PodCheckDns"

func (d PodCheckDns) Name() string {
	return PodCheckDnsName
}
func (d PodCheckDns) Description() string {
	return "Check that DNS within a pod works as expected"
}
func (d PodCheckDns) CanRun() (bool, error) {
	return true, nil
}

func (d PodCheckDns) Check() types.DiagnosticResult {
	r := types.NewDiagnosticResult(PodCheckDnsName)

	conf, err := ioutil.ReadFile(filename)
	if err != nil {
		r.Error("DP2001", err, fmt.Sprintf("could not load resolver file /etc/resolv.conf: %v", err))
		return r
	}
	nsFinder := regexp.MustCompile("^\\s*nameserver\\s+(\\w+)")
	servers = nsFinder.FindStringSubmatch(conf)
	if servers == nil {
		r.Error("DP2002", nil, "could not find any nameservers defined in /etc/resolv.conf")
		return r
	}

	for index, server := range servers {
		failure := false
		// put together a DNS query to known nameservers for kubernetes.default
		msg := new(dns.Msg)
		msg.SetQuestion("kubernetes.default.svc.cluster.local.", dns.TypeA)
		msg.RecursionDesired = false
		if in, err := dns.Exchange(m1, server+":53"); err != nil {
			if index == 0 {
				r.Error("DP2003", err, fmt.Sprintf("The first /etc/resolv.conf nameserver %s\ncould not resolve kubernetes.default.svc.cluster.local.\nError: %v\nThis nameserver points to the master's SkyDNS which is critical for\nresolving cluster names, e.g. for Services.", server, err))
			} else {
				r.Warn("DP2004", err, fmt.Sprintf("error querying nameserver %s:\n  %v", server, err))
			}
			continue
		} else {
			r.Debug("DP2005", fmt.Sprintf("successful query to nameserver %s", server))
		}
	}

	return r
}
