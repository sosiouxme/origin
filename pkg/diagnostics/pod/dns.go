package pod

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"time"

	"github.com/miekg/dns"
	"github.com/openshift/origin/pkg/diagnostics/types"
)

const PodCheckDnsName = "PodCheckDns"

// PodCheckDns is a Diagnostic to check that DNS within a pod works as expected
type PodCheckDns struct {
}

type dnsResponse struct {
	in  *dns.Msg
	err error
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
		// put together a DNS query to configured nameservers for kubernetes.default
		msg := new(dns.Msg)
		msg.SetQuestion("kubernetes.default.svc.cluster.local.", dns.TypeA)
		msg.RecursionDesired = false
		rchan := make(chan dnsResponse, 1)
		go func() {
			in, err := dns.Exchange(msg, server[1]+":53")
			rchan <- dnsResponse{in, err}
		}()
		select {
		case <-time.After(time.Second * 2):
			if serverIndex == 0 { // in a pod, master (SkyDNS) IP is injected as first nameserver
				r.Warn("DP2009", nil, fmt.Sprintf("A request to the master (SkyDNS) nameserver %s timed out.\nThis could be temporary but could also indicate network or DNS problems.\nThis nameserver is critical for resolving cluster DNS names.", server[1]))
			} else {
				r.Warn("DP2010", nil, fmt.Sprintf("A request to the nameserver %s timed out.\nThis could be temporary but could also indicate network or DNS problems.", server[1]))
			}
		case result := <-rchan:
			in, err := result.in, result.err
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
	}

	return r
}
