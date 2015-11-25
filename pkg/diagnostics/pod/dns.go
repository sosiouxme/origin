package pod

import (
	"fmt"
	"io/ioutil"
	"regexp"

	"github.com/miekg/dns"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

const PodCheckDnsName = "PodCheckDns"

// PodCheckDns is a Diagnostic to check that DNS within a pod works as expected
type PodCheckDns struct {
}

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

	conf, err := ioutil.ReadFile("/etc/resolv.conf")
	if err != nil {
		r.Error("DP2001", err, fmt.Sprintf("could not load resolver file /etc/resolv.conf: %v", err))
		return r
	}
	nsFinder := regexp.MustCompile("(?im:^\\s*nameserver\\s+(\\S+))")
	servers := nsFinder.FindAllStringSubmatch(string(conf), 3)
	if servers == nil {
		r.Error("DP2002", nil, "could not find any nameservers defined in /etc/resolv.conf")
		return r
	}

	for serverIndex, server := range servers {
		// put together a DNS query to known nameservers for kubernetes.default
		msg := new(dns.Msg)
		msg.SetQuestion("kubernetes.default.svc.cluster.local.", dns.TypeA)
		msg.RecursionDesired = false
		in, err := dns.Exchange(msg, server[1]+":53")
		//fmt.Printf("Result is %#v\n", in)
		//fmt.Printf("Answer section is %#v", in.Answer)
		if serverIndex == 0 { // in a pod, master (SkyDNS) IP is injected as first nameserver
			if err != nil {
				r.Error("DP2003", err, fmt.Sprintf("The first /etc/resolv.conf nameserver %s\ncould not resolve kubernetes.default.svc.cluster.local.\nError: %v\nThis nameserver points to the master's SkyDNS which is critical for\nresolving cluster names, e.g. for Services.", server[1], err))
			} else if len(in.Answer) == 0 {
				r.Error("DP2006", err, fmt.Sprintf("The first /etc/resolv.conf nameserver %s\ncould not resolve kubernetes.default.svc.cluster.local.\nReturn code: %v\nThis nameserver points to the master's SkyDNS which is critical for\nresolving cluster names, e.g. for Services.", server[1], dns.RcodeToString[in.MsgHdr.Rcode]))
			} else {
				r.Debug("DP2007", fmt.Sprintf("The first /etc/resolv.conf nameserver %s\nresolved kubernetes.default.svc.cluster.local. to:\n  %s", server[1], in.Answer[0]))
			}
		} else if err != nil {
			r.Warn("DP2004", err, fmt.Sprintf("Error querying nameserver %s:\n  %v\nThis may indicate a problem with non-cluster DNS.", server[1], err))
		} else {
			rcode := in.MsgHdr.Rcode
			switch rcode {
			case dns.RcodeSuccess, dns.RcodeNameError: // aka NXDOMAIN
				r.Debug("DP2005", fmt.Sprintf("Successful query to nameserver %s", server[1]))
			default:
				r.Warn("DP2008", nil, fmt.Sprintf("Received unexpected return code '%s' from nameserver %s:\nThis may indicate a problem with non-cluster DNS.", dns.RcodeToString[rcode], server[1]))
			}
		}
	}

	return r
}
